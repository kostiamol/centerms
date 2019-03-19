package api

import (
	"encoding/json"
	"net/http"

	"github.com/kostiamol/centerms/store"

	"github.com/Sirupsen/logrus"
)

func respMeta(w http.ResponseWriter, meta interface{}) {
	resp := map[string]interface{}{"meta": meta}
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(b); err != nil {
		logrus.Error(err)
	}
}

func resp(w http.ResponseWriter, data interface{}, md ...store.Meta) {
	resp := map[string]interface{}{"data": data}

	meta := map[string]interface{}{}
	for _, mt := range md {
		for key, m := range mt.GetMeta() {
			meta[key] = m
		}
	}

	if len(meta) > 0 {
		resp["meta"] = meta
	}

	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(b); err != nil {
		logrus.Error(err)
	}
}

func respError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	code := http.StatusInternalServerError
	resp := map[string]interface{}{
		"code":    ErrService,
		"message": "Internet Server Error",
	}

	switch apiErr := err.(type) {
	case apiError:
		resp["code"] = apiErr.Code
		resp["message"] = apiErr.Message

		switch apiErr.Code {
		case ErrNotFound:
			code = http.StatusNotFound
		case ErrBadRequest, ErrBadParam:
			code = http.StatusBadRequest
		case ErrAuth, ErrBadJwt:
			code = http.StatusUnauthorized
		}
	case validationError:
		globalErr := apiErr.GlobalMessage
		if globalErr == nil {
			globalErr = []string{}
		}

		resp["code"] = apiErr.Code
		resp["message"] = apiErr.Message
		resp["validation_errors"] = map[string]interface{}{"_error": globalErr, "errors": apiErr.Errors}
	default:
		logrus.WithField("API", "jsonError").Error(err)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		logrus.Error(err)
	}

	w.WriteHeader(code)

	if _, err = w.Write(b); err != nil {
		logrus.Error(err)
	}
}
