package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kostiamol/centerms/store/model"
)

var getTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// todo
})

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		a.log.Errorf("func Write: %s", err)
	}
}

func (a *api) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	d, err := a.dataProvider.GetDevsData()
	if err != nil {
		a.log.Errorf("func getDevsDataHandler: func GetDevsData: %s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.Errorf("func getDevsDataHandler: func Encode: %s", err)
		return
	}
}

func (a *api) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	devID := mux.Vars(r)["dev_id"]
	d, err := a.dataProvider.GetDevData(devID)
	if err != nil {
		a.log.Errorf("func getDevDataHandler: func GetDevData: %s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.Errorf("func getDevDataHandler: func Encode: %s", err)
		return
	}
}

func (a *api) getDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	devID := mux.Vars(r)["dev_id"]
	t := mux.Vars(r)["type"]
	c, err := a.cfgProvider.GetCfg(devID, model.Type(t))
	if err != nil {
		a.log.Errorf("func getCfgHandler: func GetCfg: %s", err)
		return
	}
	if _, err = w.Write(c.Data); err != nil {
		a.log.Errorf("func getCfgHandler: func Write: %s", err)
		return
	}
}

func (a *api) patchDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	var c model.Cfg
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		a.log.Errorf("func patchCfgHandler: func Decode: %s", err)
		return
	}

	vars := mux.Vars(r)
	if err := a.cfgProvider.SetCfg(vars["dev_id"], model.Type(vars["type"]), &c); err != nil {
		a.log.Errorf("func patchCfgHandler: func SetCfg: %s", err)
		return
	}

	a.pubChan <- &c
}
