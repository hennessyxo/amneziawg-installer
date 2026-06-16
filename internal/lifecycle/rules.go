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
// Both an expired client and an over-quota client are *disabled* (not deleted),
// so the admin can re-enable them later. Already-disabled clients are left
// alone, so the enforcer never acts twice on the same client.
func Evaluate(r Record, now time.Time) Action {
	if r.Disabled {
		return ActionNone
	}
	if r.ExpiresAt != nil && now.After(*r.ExpiresAt) {
		return ActionDisable
	}
	if r.QuotaBytes > 0 && r.UsedBytes >= r.QuotaBytes {
		return ActionDisable
	}
	return ActionNone
}
