package db

import "github.com/google/wire"

// Set provides db connection for dependency injection.
var Set = wire.NewSet(
	NewDB,
)
