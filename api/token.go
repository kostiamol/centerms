package api

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/kostiamol/centerms/errors"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/context"
)

const (
	ownerIDKey = "ownerID"
	authHeader = "Authorization"
)

type token struct {
	publicKey *rsa.PublicKey
}

func newToken(pubKey string) (*token, error) {
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("unable to parse RSA public key: %v", err)
	}
	return &token{publicKey: publicKey}, nil
}

func (t *token) validator(next http.HandlerFunc, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := t.getTokenClaims(w, r)
		if err != nil {
			ERROR(w, errors.NewBadJwtError(err.Error()))
			return
		}

		orgId, ok := claims[ownerIDKey]
		if !ok {
			ERROR(w, errors.NewBadJwtError("empty ownerID"))
			return
		}
		orgIdStr, ok := orgId.(string)
		if !ok {
			ERROR(w, errors.NewBadJwtError("invalid ownerID"))
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), ownerIDKey, orgIdStr))
		next(w, r)
	}
}

func (t *token) getTokenClaims(w http.ResponseWriter, r *http.Request) (jwt.MapClaims, error) {
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
		return nil, fmt.Errorf("can not parse token, %v", err)
	}

	if token == nil {
		return nil, fmt.Errorf("token is empty")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}
