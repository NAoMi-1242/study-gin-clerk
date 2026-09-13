package aimodel

import "github.com/google/wire"

var Set = wire.NewSet(
	NewService,
	NewHandler,
)
