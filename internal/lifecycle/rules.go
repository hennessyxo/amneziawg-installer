package lifecycle

import "time"

// Action is what the enforcer should do with a client.
type Action int

const (
	ActionNone    Action = iota // leave as-is
	ActionDisable               // cut off from the live interface (re-enable possible)
	ActionDelete                // remove entirely
)

func (a Action) String() string {
	switch a {
	case ActionDisable:
		return "disable"
	case ActionDelete:
		return "delete"
	default:
		return "none"
	}
}

// Evaluate decides what should happen to a record at time now.
//
//   - An expired client is deleted (the "config for N days, then gone" case).
//   - A client over its quota is disabled (kept, so the admin can re-enable or
//     reset it) — deleting would throw away the config.
func Evaluate(r Record, now time.Time) Action {
	if r.ExpiresAt != nil && now.After(*r.ExpiresAt) {
		return ActionDelete
	}
	if !r.Disabled && r.QuotaBytes > 0 && r.UsedBytes >= r.QuotaBytes {
		return ActionDisable
	}
	return ActionNone
}
