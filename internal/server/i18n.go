package server

import "net/http"

const (
	defaultLang = "ru"
	langCookie  = "lang"
)

// langs holds the UI string catalog per language. Keys are referenced from the
// templates as {{.L.key}} and from a few server-side strings.
var langs = map[string]map[string]string{
	"ru": {
		"login_sub":   "Управление self-hosted VPN",
		"login_pw":    "Пароль администратора",
		"login_btn":   "Войти",
		"login_err":   "Неверный пароль",
		"logout":      "Выйти",
		"clients":     "Клиенты",
		"ph_name":     "имя клиента",
		"ph_days":     "срок, дней",
		"ph_quota":    "квота, ГБ",
		"ph_speed":    "скорость, Мбит/с",
		"add":         "+ Добавить",
		"online":      "онлайн",
		"h_client":    "Клиент",
		"h_speed_dn":  "↓ скорость",
		"h_speed_up":  "↑ скорость",
		"h_limit":     "лимит",
		"h_quota":     "квота",
		"h_expiry":    "срок",
		"h_handshake": "handshake",
		"disabled":    "отключён",
		"conf":        "конфиг",
		"on":          "вкл",
		"off":         "выкл",
		"confirm_del": "Удалить клиента",
		"no_clients":  "пока нет клиентов — добавь первого выше",
		"created_msg": "создан — отсканируй QR в приложении AmneziaVPN или скачай конфиг",
		"download":    "Скачать .conf",
		"expired":     "истёк",
		"speed_unit":  "Мбит/с",
	},
	"en": {
		"login_sub":   "Self-hosted VPN management",
		"login_pw":    "Admin password",
		"login_btn":   "Sign in",
		"login_err":   "Wrong password",
		"logout":      "Sign out",
		"clients":     "Clients",
		"ph_name":     "client name",
		"ph_days":     "days",
		"ph_quota":    "quota, GB",
		"ph_speed":    "speed, Mbit/s",
		"add":         "+ Add",
		"online":      "online",
		"h_client":    "Client",
		"h_speed_dn":  "↓ speed",
		"h_speed_up":  "↑ speed",
		"h_limit":     "limit",
		"h_quota":     "quota",
		"h_expiry":    "expires",
		"h_handshake": "handshake",
		"disabled":    "disabled",
		"conf":        "conf",
		"on":          "on",
		"off":         "off",
		"confirm_del": "Delete client",
		"no_clients":  "no clients yet — add the first one above",
		"created_msg": "created — scan the QR in AmneziaVPN or download the config",
		"download":    "Download .conf",
		"expired":     "expired",
		"speed_unit":  "Mbit/s",
	},
}

// normLang clamps an arbitrary value to a supported language code.
func normLang(s string) string {
	if _, ok := langs[s]; ok {
		return s
	}
	return defaultLang
}

// tr returns the string catalog for a language.
func tr(lang string) map[string]string { return langs[normLang(lang)] }

// lang reads the preferred language from the cookie (default Russian).
func (s *Server) lang(r *http.Request) string {
	if c, err := r.Cookie(langCookie); err == nil {
		return normLang(c.Value)
	}
	return defaultLang
}

// setLang stores the language preference and redirects back.
func (s *Server) setLang(w http.ResponseWriter, r *http.Request) {
	code := normLang(r.PathValue("code"))
	http.SetCookie(w, &http.Cookie{
		Name:     langCookie,
		Value:    code,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   365 * 24 * 3600,
	})
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
