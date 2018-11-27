package svc

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/entity"
)

var mySigningKey = []byte("secret")

func (w *Web) redirectHandler(rw http.ResponseWriter, r *http.Request) {
	newURI := "https://" + r.Host + r.URL.String()
	http.Redirect(rw, r, newURI, http.StatusFound)
}

var getTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// create the token
	token := jwt.New(jwt.SigningMethodHS256)

	// create a map to store the claims
	claims := token.Claims.(jwt.MapClaims)

	// sett token claims
	claims["admin"] = true
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	// sign the token with the secret
	tokenString, _ := token.SignedString(mySigningKey)

	// finally, write the token to the browser window
	w.Write([]byte(tokenString))
})

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

func (w *Web) getDevsDataHandler(rw http.ResponseWriter, r *http.Request) {
	conn, err := w.store.CreateConn()
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	d, err := conn.GetDevsData()
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(rw).Encode(d); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (w *Web) getDevDataHandler(rw http.ResponseWriter, r *http.Request) {
	conn, err := w.store.CreateConn()
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entity.DevID(mux.Vars(r)["id"])
	d, err := conn.GetDevData(id)
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(rw).Encode(d); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (w *Web) getDevCfgHandler(rw http.ResponseWriter, r *http.Request) {
	conn, err := w.store.CreateConn()
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entity.DevID(mux.Vars(r)["id"])
	c, err := conn.GetDevCfg(id)
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevCfgHandler",
		}).Errorf("%s", err)
		return
	}

	if _, err = rw.Write(c.Data); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "getDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
}

func (w *Web) patchDevCfgHandler(rw http.ResponseWriter, r *http.Request) {
	conn, err := w.store.CreateConn()
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	var c entity.DevCfg
	if err = json.NewDecoder(r.Body).Decode(&c); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}

	id := entity.DevID(mux.Vars(r)["id"])
	if err = conn.SetDevCfg(entity.DevID(id), &c); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}

	b, err := json.Marshal(c)
	if err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err = conn.Publish(b, w.pubChan); err != nil {
		w.log.WithFields(logrus.Fields{
			"func": "patchDevCfgHandler",
		}).Errorf("%s", err)
		return
	}
}
