package tgbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/hennessyxo/amneziawg-installer/internal/awgctl"
)

// VPN is the subset of the AmneziaWG controller the bot needs. awgctl.FileController
// satisfies it; tests use a fake.
type VPN interface {
	AddClient(name string, opts awgctl.AddOptions) (awgctl.Client, error)
	RevokeClient(name string) error
	ClientConfig(name string) (string, error)
	ServerClients() ([]awgctl.ServerClient, error)
}

// Bot wires the Telegram API, the VPN controller and the access gate together.
type Bot struct {
	api   *API
	vpn   VPN
	auth  *Auth
	iface string
	lang  string
}

// New builds a Bot. lang is "ru" or "en" for its replies.
func New(api *API, vpn VPN, a *Auth, iface, lang string) *Bot {
	if lang != "en" {
		lang = "ru"
	}
	return &Bot{api: api, vpn: vpn, auth: a, iface: iface, lang: lang}
}

const pollTimeout = 30 // seconds for long polling

// Run polls Telegram for updates until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	var offset int64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		updates, err := b.api.GetUpdates(offset, pollTimeout)
		if err != nil {
			log.Printf("awg-bot: getUpdates: %v", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil && u.Message.Text != "" {
				b.handle(u.Message)
			}
		}
	}
}

// handle dispatches a single message.
func (b *Bot) handle(m *Message) {
	cmd, ok := ParseCommand(m.Text)
	if !ok {
		return // ignore non-command chatter
	}
	var senderID int64
	if m.From != nil {
		senderID = m.From.ID
	}
	chatID := m.Chat.ID

	switch cmd.Name {
	case "start", "help":
		b.reply(chatID, b.helpText(senderID))
		return
	case "auth":
		b.handleAuth(chatID, senderID, m.MessageID, cmd.Args)
		return
	}

	// Everything below requires authorization (allowlist AND password).
	if !b.auth.IsAuthorized(senderID) {
		b.reply(chatID, b.accessMsg(senderID))
		return
	}

	switch cmd.Name {
	case "new":
		b.handleNew(chatID, cmd.Args)
	case "list":
		b.handleList(chatID)
	case "revoke":
		b.handleRevoke(chatID, cmd.Args)
	case "config":
		b.handleConfig(chatID, cmd.Args)
	default:
		b.reply(chatID, b.t("unknown"))
	}
}

func (b *Bot) handleAuth(chatID, senderID, messageID int64, args []string) {
	b.api.DeleteMessage(chatID, messageID) // scrub the typed password from history
	if !b.auth.IsAdmin(senderID) {
		b.reply(chatID, b.t("not_allowed")) // not on the allowlist — password won't help
		return
	}
	if !b.auth.HasPassword() {
		b.reply(chatID, b.t("auth_disabled"))
		return
	}
	if len(args) == 0 {
		b.reply(chatID, b.t("auth_usage"))
		return
	}
	if b.auth.TryPassword(senderID, strings.Join(args, " ")) {
		b.reply(chatID, b.t("auth_ok"))
	} else {
		b.reply(chatID, b.t("auth_fail"))
	}
}

func (b *Bot) handleNew(chatID int64, args []string) {
	if len(args) == 0 {
		b.reply(chatID, b.t("new_usage"))
		return
	}
	name, valid := awgctl.SanitizeName(args[0])
	if !valid {
		b.reply(chatID, b.t("bad_name"))
		return
	}
	client, err := b.vpn.AddClient(name, awgctl.AddOptions{})
	if err != nil {
		b.reply(chatID, b.t("new_fail")+err.Error())
		return
	}
	b.sendProfile(chatID, client.Name, client.Config)
}

func (b *Bot) handleList(chatID int64) {
	clients, err := b.vpn.ServerClients()
	if err != nil {
		b.reply(chatID, b.t("list_fail")+err.Error())
		return
	}
	if len(clients) == 0 {
		b.reply(chatID, b.t("list_empty"))
		return
	}
	var sb strings.Builder
	sb.WriteString(b.t("list_head") + "\n")
	for _, c := range clients {
		sb.WriteString("• " + c.Name + "\n")
	}
	b.reply(chatID, sb.String())
}

func (b *Bot) handleRevoke(chatID int64, args []string) {
	if len(args) == 0 {
		b.reply(chatID, b.t("revoke_usage"))
		return
	}
	name, valid := awgctl.SanitizeName(args[0])
	if !valid {
		b.reply(chatID, b.t("bad_name"))
		return
	}
	if err := b.vpn.RevokeClient(name); err != nil {
		b.reply(chatID, b.t("revoke_fail")+err.Error())
		return
	}
	b.reply(chatID, fmt.Sprintf(b.t("revoke_ok"), name))
}

func (b *Bot) handleConfig(chatID int64, args []string) {
	if len(args) == 0 {
		b.reply(chatID, b.t("config_usage"))
		return
	}
	name, valid := awgctl.SanitizeName(args[0])
	if !valid {
		b.reply(chatID, b.t("bad_name"))
		return
	}
	conf, err := b.vpn.ClientConfig(name)
	if err != nil {
		b.reply(chatID, b.t("config_fail"))
		return
	}
	b.sendProfile(chatID, name, conf)
}

// sendProfile delivers a client's config as a .conf document plus a QR image.
func (b *Bot) sendProfile(chatID int64, name, conf string) {
	filename := fmt.Sprintf("%s-client-%s.conf", b.iface, name)
	if err := b.api.SendDocument(chatID, filename, []byte(conf), name); err != nil {
		b.reply(chatID, b.t("send_fail")+err.Error())
		return
	}
	if png, err := qrcode.Encode(conf, qrcode.Low, 512); err == nil {
		_ = b.api.SendPhoto(chatID, name+".png", png, "QR — "+name)
	}
	b.reply(chatID, b.t("profile_note"))
}

func (b *Bot) reply(chatID int64, text string) {
	if err := b.api.SendMessage(chatID, text); err != nil {
		log.Printf("awg-bot: sendMessage: %v", err)
	}
}

func (b *Bot) helpText(senderID int64) string {
	head := b.t("help")
	if !b.auth.IsAuthorized(senderID) {
		head += "\n\n" + b.accessMsg(senderID)
	}
	return head
}

// accessMsg returns the right "no access" reply: a non-admin is simply not
// allowed; an allowlisted user who hasn't entered the password is prompted to.
func (b *Bot) accessMsg(senderID int64) string {
	if !b.auth.IsAdmin(senderID) {
		return b.t("not_allowed")
	}
	return b.t("denied")
}
