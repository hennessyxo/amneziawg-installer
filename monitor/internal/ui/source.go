package ui

import "github.com/hennessyxo/amneziawg-installer/monitor/internal/awg"

// Source supplies fresh snapshots to the UI. Implementations live in main
// (a real `awg` command runner and a synthetic demo generator) so the UI stays
// decoupled from process execution and is easy to test.
type Source interface {
	Fetch() (awg.Snapshot, error)
}
