package constant

import "sync/atomic"

// setup guards system initialization. Use the accessor functions below
// instead of reading/writing this directly.
var setup atomic.Bool

// IsSetup reports whether initial setup has been completed.
func IsSetup() bool {
	return setup.Load()
}

// SetSetup marks setup as completed (used during boot-time init).
func SetSetup(v bool) {
	setup.Store(v)
}

// TryClaimSetup atomically transitions from not-setup to setup.
// Returns true if the caller won the race, false if setup was already
// claimed by another goroutine or a previous boot.
func TryClaimSetup() bool {
	return setup.CompareAndSwap(false, true)
}
