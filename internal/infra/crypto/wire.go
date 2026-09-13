package crypto

import "github.com/google/wire"

// Set provides crypto components for dependency injection.
var Set = wire.NewSet(
	NewAESCipher,
)
