package common

import (
	"context"
	"fmt"
	"runtime/debug"
)

// SafeGo starts a goroutine with panic recovery.
// If the function panics, it logs the error and recovers.
// Use this for fire-and-forget operations that don't need to be tracked.
func SafeGo(f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				SysError(fmt.Sprintf("panic in SafeGo: %v\n%s", r, debug.Stack()))
			}
		}()
		f()
	}()
}

// SafeGoWithContext starts a goroutine with panic recovery and context support.
// The function receives a context that should be checked for cancellation.
func SafeGoWithContext(ctx context.Context, f func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				SysError(fmt.Sprintf("panic in SafeGoWithContext: %v\n%s", r, debug.Stack()))
			}
		}()
		f(ctx)
	}()
}

// SafeGoNamed starts a named goroutine with panic recovery.
// The name is used in error logging for easier debugging.
func SafeGoNamed(name string, f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				SysError(fmt.Sprintf("panic in SafeGo[%s]: %v\n%s", name, r, debug.Stack()))
			}
		}()
		f()
	}()
}

// MustGo starts a goroutine with panic recovery that retries on panic.
// Use with caution - infinite retries can cause issues.
// maxRetries of 0 means no limit.
func MustGo(f func(), maxRetries int) {
	go func() {
		retries := 0
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						retries++
						SysError(fmt.Sprintf("panic in MustGo (retry %d): %v\n%s", retries, r, debug.Stack()))
					}
				}()
				f()
			}()

			// If we reach here without panic, we're done
			if maxRetries > 0 && retries >= maxRetries {
				SysError(fmt.Sprintf("MustGo exceeded max retries (%d)", maxRetries))
				return
			}

			// If no panic occurred, exit the loop
			return
		}
	}()
}
