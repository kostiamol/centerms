package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/entities"
)

var mySigningKey = []byte("secret")

func (s *WebService) redirectHandler(w http.ResponseWriter, r *http.Request) {
	newURI := "https://" + r.Host + r.URL.String()
	http.Redirect(w, r, newURI, http.StatusFound)
}

var getTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	/* Create the token */
	token := jwt.New(jwt.SigningMethodHS256)

	// Create a map to store our claims
	claims := token.Claims.(jwt.MapClaims)

	/* Set token claims */
	claims["admin"] = true
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	/* Sign the token with our secret */
	tokenString, _ := token.SignedString(mySigningKey)

	/* Finally, write the token to the browser window */
	w.Write([]byte(tokenString))
})

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

func (s *WebService) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	data, err := conn.GetDevsData()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevsDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entities.DevID(mux.Vars(r)["id"])
	data, err := conn.GetDevData(id)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}

	if err = json.NewEncoder(w).Encode(data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevDataHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) getDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	id := entities.DevID(mux.Vars(r)["id"])
	config, err := conn.GetDevConfig(id)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	if _, err = w.Write(config.Data); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "getDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}

func (s *WebService) patchDevConfigHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.storage.CreateConn()
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	defer conn.CloseConn()

	var config entities.DevConfig
	if err = json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	id := entities.DevID(mux.Vars(r)["id"])
	if err = conn.SetDevConfig(entities.DevID(id), &config); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
	if _, err = conn.Publish(b, s.pubChan); err != nil {
		s.log.WithFields(logrus.Fields{
			"func": "patchDevConfigHandler",
		}).Errorf("%s", err)
		return
	}
}
