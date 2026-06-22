// Command awg-bot is an access-controlled Telegram bot for issuing AmneziaWG
// client profiles. It runs on the server next to the VPN, polls Telegram (no
// inbound port needed) and answers a small set of commands (/new, /list,
// /revoke, /config) for authorized users only.
//
// Usage:
//
//	awg-bot --token-file /etc/amnezia/amneziawg/bot.token \
//	        --password-hash-file /etc/amnezia/amneziawg/bot.hash \
//	        --admins 12345678,87654321
//
//	echo 'mysecret' | awg-bot hash    # print a bcrypt hash for the installer
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
	"github.com/hennessyxo/amneziawg-installer/internal/awgctl"
	"github.com/hennessyxo/amneziawg-installer/internal/lifecycle"
	"github.com/hennessyxo/amneziawg-installer/internal/tgbot"
)

func main() {
	// Subcommand: `awg-bot hash` reads a password from stdin and prints its bcrypt
	// hash, so the installer can store a hash instead of the plaintext password.
	if len(os.Args) > 1 && os.Args[1] == "hash" {
		makeHash()
		return
	}

	token := flag.String("token", "", "Telegram bot token (prefer --token-file)")
	tokenFile := flag.String("token-file", "", "file containing the Telegram bot token")
	admins := flag.String("admins", "", "comma-separated Telegram user IDs always allowed")
	hashFile := flag.String("password-hash-file", "", "file with the access-password bcrypt hash")
	authStore := flag.String("auth-store", "/etc/amnezia/amneziawg/bot-authorized.json", "where authorized chats are remembered")
	iface := flag.String("iface", "awg0", "AmneziaWG interface")
	conf := flag.String("conf", "/etc/amnezia/amneziawg/awg0.conf", "server config path")
	params := flag.String("params", "/etc/amnezia/amneziawg/params", "installer params path")
	clientDir := flag.String("client-dir", "/etc/amnezia/amneziawg/clients", "where client .conf files are stored")
	storePath := flag.String("store", "/etc/amnezia/amneziawg/clients.json", "lifecycle metadata store")
	lang := flag.String("lang", "ru", "bot reply language (ru|en)")
	flag.Parse()

	tok, err := loadToken(*token, *tokenFile)
	if err != nil {
		log.Fatalf("awg-bot: %v", err)
	}

	var pwHash string
	if *hashFile != "" {
		b, err := os.ReadFile(*hashFile)
		if err != nil {
			log.Fatalf("awg-bot: reading password hash: %v", err)
		}
		pwHash = strings.TrimSpace(string(b))
	}

	adminIDs, err := parseAdmins(*admins)
	if err != nil {
		log.Fatalf("awg-bot: %v", err)
	}
	if len(adminIDs) == 0 && pwHash == "" {
		log.Fatalf("awg-bot: no access configured — set --admins and/or --password-hash-file")
	}

	store, err := lifecycle.Open(*storePath)
	if err != nil {
		log.Fatalf("awg-bot: %v", err)
	}
	ctrl := awgctl.FileController{
		Iface:     *iface,
		ConfPath:  *conf,
		ParamPath: *params,
		ClientDir: *clientDir,
		Store:     store,
	}

	gate := tgbot.NewAuth(adminIDs, pwHash, *authStore)
	bot := tgbot.New(tgbot.NewAPI(tok), ctrl, gate, *iface, *lang)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("awg-bot: started (iface %s, %d admin(s), password=%v)", *iface, len(adminIDs), pwHash != "")
	if err := bot.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("awg-bot: %v", err)
	}
}

// loadToken reads the bot token from a file (preferred) or the flag.
func loadToken(token, tokenFile string) (string, error) {
	if tokenFile != "" {
		b, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", fmt.Errorf("reading token file: %w", err)
		}
		token = string(b)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("no bot token (use --token-file or --token)")
	}
	return token, nil
}

// parseAdmins parses a comma-separated list of Telegram user IDs.
func parseAdmins(s string) ([]int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var ids []int64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid admin id %q", part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// makeHash reads one line from stdin and prints its bcrypt hash.
func makeHash() {
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		log.Fatal("awg-bot hash: no password on stdin")
	}
	hash, err := auth.HashPassword(strings.TrimSpace(sc.Text()))
	if err != nil {
		log.Fatalf("awg-bot hash: %v", err)
	}
	fmt.Println(hash)
}
