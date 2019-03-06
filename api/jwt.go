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
	publicKey           *rsa.PublicKey
	authorizationHeader string
}

// NewJwtFromBase64 initialises token structure by base64-encoded pem-key
func newJWT(pubKey string) (*token, error) {
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("unable to parse RSA public key: %v", err)
	}
	return &token{publicKey: publicKey}, nil
}

func (t *token) jwtValidator(next http.HandlerFunc, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get(authHeader), " ")
		if len(authHeader) < 2 {
			ERROR(w, errors.NewBadJwtError(fmt.Sprintf("bad %s header", authHeader)))
			return
		}
		parsedJWT, err := jwt.Parse(authHeader[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return t.publicKey, nil
		})
		if err != nil {
			if e, ok := err.(*jwt.ValidationError); ok {
				ERROR(w, errors.NewBadJwtError(e.Error()))
				return
			}
			ERROR(w, errors.NewBadJwtError(err.Error()))
			return
		}

		if parsedJWT == nil {
			ERROR(w, errors.NewBadJwtError("parsedJWT is empty"))
			return
		}

		claim, ok := parsedJWT.Claims.(jwt.MapClaims)
		if !ok || !parsedJWT.Valid {
			ERROR(w, errors.NewBadJwtError("parsedJWT is invalid"))
			return
		}
		orgId, ok := claim[ownerIDKey]
		if !ok {
			ERROR(w, errors.NewBadJwtError("empty orgId"))
			return
		}
		orgIdStr, ok := orgId.(string)
		if !ok {
			ERROR(w, errors.NewBadJwtError("invalid orgId"))
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), ownerIDKey, orgIdStr))
		next(w, r)
	}
}
