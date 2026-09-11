package router

import "github.com/google/wire"

// Set provides router dependency injection.
var Set = wire.NewSet(
	wire.Struct(new(Dependencies), "*"),
	New,
)

