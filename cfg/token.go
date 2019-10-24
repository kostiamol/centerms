package cfg

import (
	"fmt"
)

// Token holds keys for token handling.
type Token struct {
	PublicKey  string
	PrivateKey string
}

func (t Token) Validate() error {
	if t.PublicKey == "" {
		return fmt.Errorf("public key env var is missing")
	}
	if t.PrivateKey == "" {
		return fmt.Errorf("private key env var is missing")
	}
	return nil
}
