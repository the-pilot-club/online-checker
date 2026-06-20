// Package store abstracts persistence of online session state so the backend
// (currently Redis) can be swapped out without touching the checker logic.
package store

import "context"

// SessionStore persists session state keyed by a VATSIM CID.
//
// Implementations must distinguish a genuinely missing key (found == false,
// err == nil) from a backend failure (err != nil) so callers never mistake an
// outage for an absent session.
type SessionStore[T any] interface {
	// Get returns the stored value for cid. found is false when no value
	// exists; err is non-nil only on a backend failure.
	Get(ctx context.Context, cid string) (value T, found bool, err error)
	// Set stores value for cid, refreshing its expiry.
	Set(ctx context.Context, cid string, value T) error
	// Delete removes any stored value for cid.
	Delete(ctx context.Context, cid string) error
	// Close releases the underlying resources.
	Close() error
}
