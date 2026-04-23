package errorx

import "errors"

var (
	ErrHostNotFound    = errors.New("virtual-machine: host not found")
	ErrHostInvalid     = errors.New("virtual-machine: host invalid")
	ErrHostConflict    = errors.New("virtual-machine: host conflict")
	ErrHostForbidden   = errors.New("virtual-machine: host forbidden")
	ErrHostUnavailable = errors.New("virtual-machine: host unavailable")
)
