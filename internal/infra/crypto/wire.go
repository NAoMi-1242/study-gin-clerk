package crypto

import (
	"github.com/google/wire"

	"study-gin-clerk/internal/config"
)

// ProvideCipher creates a concrete *AESCipher using the application configuration.
func ProvideCipher(cfg config.Config) (*AESCipher, error) {
	return NewAESCipher(cfg.EncryptionKey)
}

// Set provides crypto components for dependency injection.
var Set = wire.NewSet(
	ProvideCipher,
	wire.Bind(new(Cipher), new(*AESCipher)),
)
