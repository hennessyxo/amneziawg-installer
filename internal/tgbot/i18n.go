package tgbot

// t returns a localized bot reply for the given key.
func (b *Bot) t(key string) string {
	if m, ok := messages[b.lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return messages["en"][key]
}

var messages = map[string]map[string]string{
	"ru": {
		"help": "🤖 Бот выдачи профилей AmneziaWG.\n\n" +
			"/new <имя> — создать клиента (пришлю .conf + QR)\n" +
			"/list — список клиентов\n" +
			"/config <имя> — прислать конфиг и QR снова\n" +
			"/revoke <имя> — удалить клиента",
		"not_allowed":   "⛔ У тебя нет доступа к этому боту.",
		"denied":        "🔒 Нужна авторизация: /auth <пароль>",
		"unknown":       "Не знаю такую команду. /help",
		"auth_disabled": "Авторизация по паролю отключена.",
		"auth_usage":    "Использование: /auth <пароль>",
		"auth_ok":       "✅ Доступ открыт.",
		"auth_fail":     "❌ Неверный пароль.",
		"new_usage":     "Использование: /new <имя>",
		"bad_name":      "Некорректное имя (буквы, цифры, дефис/подчёркивание; до 32 символов).",
		"new_fail":      "Не удалось создать клиента: ",
		"list_fail":     "Не удалось получить список: ",
		"list_empty":    "Клиентов пока нет.",
		"list_head":     "Клиенты:",
		"revoke_usage":  "Использование: /revoke <имя>",
		"revoke_fail":   "Не удалось удалить: ",
		"revoke_ok":     "🗑 Клиент «%s» удалён.",
		"config_usage":  "Использование: /config <имя>",
		"config_fail":   "Конфиг клиента не найден.",
		"send_fail":     "Не удалось отправить файл: ",
		"profile_note":  "Открой приложение AmneziaWG и импортируй .conf или отсканируй QR.",
	},
	"en": {
		"help": "🤖 AmneziaWG profile bot.\n\n" +
			"/new <name> — create a client (I'll send the .conf + QR)\n" +
			"/list — list clients\n" +
			"/config <name> — resend a client's config and QR\n" +
			"/revoke <name> — remove a client",
		"not_allowed":   "⛔ You're not authorized to use this bot.",
		"denied":        "🔒 Authentication required: /auth <password>",
		"unknown":       "Unknown command. /help",
		"auth_disabled": "Password authentication is disabled.",
		"auth_usage":    "Usage: /auth <password>",
		"auth_ok":       "✅ Access granted.",
		"auth_fail":     "❌ Wrong password.",
		"new_usage":     "Usage: /new <name>",
		"bad_name":      "Invalid name (letters, digits, dash/underscore; up to 32 chars).",
		"new_fail":      "Could not create the client: ",
		"list_fail":     "Could not fetch the list: ",
		"list_empty":    "No clients yet.",
		"list_head":     "Clients:",
		"revoke_usage":  "Usage: /revoke <name>",
		"revoke_fail":   "Could not remove: ",
		"revoke_ok":     "🗑 Client \"%s\" removed.",
		"config_usage":  "Usage: /config <name>",
		"config_fail":   "Client config not found.",
		"send_fail":     "Could not send the file: ",
		"profile_note":  "Open the AmneziaWG app and import the .conf or scan the QR.",
	},
}
