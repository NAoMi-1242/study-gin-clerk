package profile

import "errors"

// ErrNotFound indicates that the user profile was not found.
var ErrNotFound = errors.New("user profile not found")
