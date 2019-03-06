package api

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"gitlab-il.cyren.io/apollo/incident-management/dao/meta"
	"gitlab-il.cyren.io/apollo/incident-management/errors"
)

// JSONMeta writes to ResponseWriter
func JSONMeta(w http.ResponseWriter, meta interface{}) {
	res := map[string]interface{}{"meta": meta}
	js, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		logrus.Error(err)
	}
}

// JSON writes to ResponseWriter a single JSON-object
func JSON(w http.ResponseWriter, data interface{}, md ...meta.Meta) {
	res := map[string]interface{}{"data": data}

	metaData := map[string]interface{}{}
	for _, mt := range md {
		for key, m := range mt.GetMetaData() {
			metaData[key] = m
		}
	}

	if len(metaData) > 0 {
		res["meta"] = metaData
	}

	js, err := json.Marshal(res)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		logrus.Error(err)
	}
}

// ERROR writes to ResponseWriter error
func ERROR(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	code := http.StatusInternalServerError
	resp := map[string]interface{}{
		"code":    errors.ErrService,
		"message": "Internet Server Error",
	}

	switch apiErr := err.(type) {
	case errors.APIError:
		resp["code"] = apiErr.Code
		resp["message"] = apiErr.Message

		switch apiErr.Code {
		case errors.ErrNotFound:
			code = http.StatusNotFound
		case errors.ErrBadRequest, errors.ErrBadParam:
			code = http.StatusBadRequest
		case errors.ErrAuth, errors.ErrBadJwt:
			code = http.StatusUnauthorized
		}
	case errors.ValidationError:

		globalErr := apiErr.GlobalMessage
		if globalErr == nil {
			globalErr = []string{}
		}

		resp["code"] = apiErr.Code
		resp["message"] = apiErr.Message
		resp["validation_errors"] = map[string]interface{}{"_error": globalErr, "errors": apiErr.Errors}
	default:
		logrus.WithField("API", "ERROR").Error(err)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		logrus.Error(err)
	}

	w.WriteHeader(code)
	_, err = w.Write(js)
	if err != nil {
		logrus.Error(err)
	}
}
