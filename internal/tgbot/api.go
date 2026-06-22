// Package tgbot implements an access-controlled Telegram bot for issuing and
// managing AmneziaWG client profiles. The transport is plain HTTP against the
// Telegram Bot API (long polling) so the server needs no inbound port; the pure
// command-parsing and authorization logic is unit-tested.
package tgbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// API is a minimal Telegram Bot API client.
type API struct {
	base string
	http *http.Client
}

// NewAPI builds a client for the given bot token.
func NewAPI(token string) *API {
	return &API{
		base: "https://api.telegram.org/bot" + token,
		// Timeout must exceed the long-poll timeout used in GetUpdates.
		http: &http.Client{Timeout: 50 * time.Second},
	}
}

// Update is one entry from getUpdates.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message is a received Telegram message (only the fields we use).
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// User is the sender of a message.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Chat is the conversation a message belongs to.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type updatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// GetUpdates long-polls for new updates starting at offset.
func (a *API) GetUpdates(offset int64, timeoutSec int) ([]Update, error) {
	q := url.Values{}
	q.Set("offset", strconv.FormatInt(offset, 10))
	q.Set("timeout", strconv.Itoa(timeoutSec))
	q.Set("allowed_updates", `["message"]`)
	resp, err := a.http.Get(a.base + "/getUpdates?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out updatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram getUpdates not ok")
	}
	return out.Result, nil
}

// SendMessage sends a plain-text message to a chat.
func (a *API) SendMessage(chatID int64, text string) error {
	q := url.Values{}
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	q.Set("text", text)
	resp, err := a.http.PostForm(a.base+"/sendMessage", q)
	if err != nil {
		return err
	}
	return drain(resp)
}

// SendDocument uploads a file (e.g. a .conf) to a chat.
func (a *API) SendDocument(chatID int64, filename string, content []byte, caption string) error {
	return a.upload("/sendDocument", "document", chatID, filename, content, caption)
}

// SendPhoto uploads an image (e.g. a QR PNG) to a chat.
func (a *API) SendPhoto(chatID int64, filename string, content []byte, caption string) error {
	return a.upload("/sendPhoto", "photo", chatID, filename, content, caption)
}

// DeleteMessage best-effort removes a message (used to scrub a typed password).
func (a *API) DeleteMessage(chatID, messageID int64) {
	q := url.Values{}
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	q.Set("message_id", strconv.FormatInt(messageID, 10))
	if resp, err := a.http.PostForm(a.base+"/deleteMessage", q); err == nil {
		_ = drain(resp)
	}
}

// upload posts a multipart form with one file field.
func (a *API) upload(method, field string, chatID int64, filename string, content []byte, caption string) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = mw.WriteField("caption", caption)
	}
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(content); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	resp, err := a.http.Post(a.base+method, mw.FormDataContentType(), &buf)
	if err != nil {
		return err
	}
	return drain(resp)
}

// drain reads and closes a response body, returning an error on non-2xx.
func drain(resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram api %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
