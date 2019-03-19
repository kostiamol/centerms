package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kostiamol/centerms/svc"

	"github.com/Sirupsen/logrus"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

var mySigningKey = []byte("secret")

var getTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	claims["admin"] = true
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	// sign the tokenValidator with the secret
	tokenString, err := token.SignedString(mySigningKey)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "getTokenHandler",
		}).Errorf("%s", err)
	}

	if _, err := w.Write([]byte(tokenString)); err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "getTokenHandler",
		}).Errorf("%s", err)
	}
})

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (a *API) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	d, err := a.dataProvider.GetDevsData()
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (a *API) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	d, err := a.dataProvider.GetDevData(id)
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (a *API) getDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	c, err := a.cfgProvider.GetDevCfg(id)
	if err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err = w.Write(c.Data); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "getDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
}

func (a *API) patchDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	var c svc.DevCfg
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := a.cfgProvider.SetDevCfg(id, &c); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err := a.cfgProvider.PublishCfgPatch(&c, a.pubChan); err != nil {
		a.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
}
