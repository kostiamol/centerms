package api

import (
	"encoding/json"
	"net/http"

	"github.com/kostiamol/centerms/log"

	"github.com/kostiamol/centerms/store/meta"
)

func respMeta(w http.ResponseWriter, meta interface{}) error { // nolint
	resp := map[string]interface{}{"meta": meta}
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(b); err != nil {
		return err
	}

	return nil
}

func resp(w http.ResponseWriter, data interface{}, md ...meta.Meta) error { // nolint
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
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(b); err != nil {
		return err
	}

	return nil
}

func respError(w http.ResponseWriter, err error, l log.Logger) {
	w.Header().Set("Content-Type", "application/json")

	code := http.StatusInternalServerError
	resp := map[string]interface{}{
		"code":    errService,
		"message": "Internet Server Error",
	}

	switch apiErr := err.(type) {
	case apiError:
		resp["code"] = apiErr.Code
		resp["message"] = apiErr.Message

		switch apiErr.Code {
		case errNotFound:
			code = http.StatusNotFound
		case errBadRequest, errBadParam:
			code = http.StatusBadRequest
		case errAuth, errBadJWT:
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
		l.With("API", "jsonError").Error(err)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		l.Error(err)
	}

	w.WriteHeader(code)

	if _, err = w.Write(b); err != nil {
		l.Error(err)
	}
}
