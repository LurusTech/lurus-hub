package catalog_sync

import "errors"

var (
	ErrUnsupportedChannelType = errors.New("catalog_sync: unsupported channel type")
	ErrCircuitBreaker         = errors.New("catalog_sync: circuit breaker tripped")
	ErrSourceVersionMismatch  = errors.New("catalog_sync: source_version mismatch; preview again")
	ErrEmptySelection         = errors.New("catalog_sync: empty selection")
)
