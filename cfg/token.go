package cfg

import (
	"fmt"
)

// Token holds keys for token handling.
type Token struct {
	PublicKey  string
	PrivateKey string
}

func (t Token) validate() error {
	if t.PublicKey == "" {
		return fmt.Errorf("PublicKey is missing")
	}
	if t.PrivateKey == "" {
		return fmt.Errorf("PrivateKey is missing")
	}
	return nil
}
