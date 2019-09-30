package api

import (
	"encoding/json"
	"net/http"

	"github.com/kostiamol/centerms/svc"

	"github.com/gorilla/mux"
)

var getTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// todo
})

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		a.log.Errorf("Write(): %s", err)
	}
}

func (a *api) getDevsDataHandler(w http.ResponseWriter, r *http.Request) {
	d, err := a.dataProvider.GetDevsData()
	if err != nil {
		a.log.Errorf("getDevsDataHandler(): GetDevsData() %s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.Errorf("getDevsDataHandler(): Encode(): %s", err)
		return
	}
}

func (a *api) getDevDataHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	d, err := a.dataProvider.GetDevData(id)
	if err != nil {
		a.log.Errorf("getDevDataHandler(): GetDevData(): %s", err)
		return
	}
	if err = json.NewEncoder(w).Encode(d); err != nil {
		a.log.Errorf("getDevDataHandler(): Encode(): %s", err)
		return
	}
}

func (a *api) getDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	c, err := a.cfgProvider.GetDevCfg(id)
	if err != nil {
		a.log.Errorf("getDevCfgHandler(): GetDevCfg(): %s", err)
		return
	}
	if _, err = w.Write(c.Data); err != nil {
		a.log.Errorf("getDevCfgHandler(): Write(): %s", err)
		return
	}
}

func (a *api) patchDevCfgHandler(w http.ResponseWriter, r *http.Request) {
	var c svc.DevCfg
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		a.log.Errorf("patchDevCfgHandler(): Decode(): %s", err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := a.cfgProvider.SetDevCfg(id, &c); err != nil {
		a.log.Errorf("patchDevCfgHandler(): SetDevCfg(): %s", err)
		return
	}
	if _, err := a.cfgProvider.PublishCfgPatch(&c, a.pubChan); err != nil {
		a.log.Errorf("patchDevCfgHandler(): PublishCfgPatch(): %s", err)
		return
	}
}
