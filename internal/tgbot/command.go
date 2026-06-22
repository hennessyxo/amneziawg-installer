package tgbot

import "strings"

// Command is a parsed bot command: a name without the leading slash and its args.
type Command struct {
	Name string
	Args []string
}

// ParseCommand extracts a command from message text. It returns ok=false when the
// text is not a command (doesn't start with "/"). A "/cmd@BotName" mention suffix
// is stripped so the bot works in groups too.
func ParseCommand(text string) (Command, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return Command{}, false
	}
	fields := strings.Fields(text)
	name := strings.TrimPrefix(fields[0], "/")
	if at := strings.IndexByte(name, '@'); at >= 0 {
		name = name[:at]
	}
	return Command{Name: strings.ToLower(name), Args: fields[1:]}, true
}
