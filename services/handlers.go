package services

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/go-swagger/go-swagger/examples/generated/models"
	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/entities"
	"github.com/kostiamol/centerms/services"
)

func (s *WebService) redirectHandler(w http.ResponseWriter, r *http.Request) {
	newURI := "https://" + r.Host + r.URL.String()
	http.Redirect(w, r, newURI, http.StatusFound)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	user := new(models.User)
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&user)
	status, token := services.Login(user)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(token)
}

func refreshToken(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	user := new(models.User)
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&user)
	w.Header().Set("Content-Type", "application/json")
	w.Write(services.RefreshToken(user))
}

func logoutHandler(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	w.Header().Set("Content-Type", "application/json")
	if err := services.Logout(r); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

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
