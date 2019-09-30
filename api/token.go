package api

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/kostiamol/centerms/log"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
)

const (
	ownerIDKey = "ownerID"
	authHeader = "Authorization"
)

type (
	tokenValidator struct {
		publicKey *rsa.PublicKey
	}
	ownerID string
)

func newTokenValidator(publicKey string) (*tokenValidator, error) {
	pk, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
	if err != nil {
		return nil, fmt.Errorf("unable to parse RSA public key: %v", err)
	}
	return &tokenValidator{publicKey: pk}, nil
}

func (t *tokenValidator) validator(next http.HandlerFunc, name string, l log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := t.getTokenClaims(w, r)
		if err != nil {
			respError(w, newBadJWTError(err.Error()), l)
			return
		}

		owner, ok := claims[ownerIDKey]
		if !ok {
			respError(w, newBadJWTError("empty ownerID"), l)
			return
		}
		ownerIDStr, ok := owner.(string)
		if !ok {
			respError(w, newBadJWTError("invalid ownerID"), l)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), ownerID(ownerIDKey), ownerIDStr))
		next(w, r)
	}
}

func (t *tokenValidator) getTokenClaims(w http.ResponseWriter, r *http.Request) (jwt.MapClaims, error) {
	authHeader := strings.Split(r.Header.Get(authHeader), " ")
	if len(authHeader) < 2 {
		return nil, fmt.Errorf(fmt.Sprintf("bad %s header", authHeader))
	}

	token, err := jwt.Parse(authHeader[1], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return t.publicKey, nil
	})

	if err != nil {
		if er, ok := err.(*jwt.ValidationError); ok {
			return nil, er
		}
		return nil, fmt.Errorf("can not parse tokenValidator, %v", err)
	}

	if token == nil {
		return nil, fmt.Errorf("tokenValidator is empty")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("tokenValidator is invalid")
	}

	return claims, nil
}
