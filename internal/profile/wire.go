package profile

import "github.com/google/wire"

var Set = wire.NewSet(
	NewRepository,
	NewService,
	NewHandler,
)

