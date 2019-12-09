package proto

import (
	"bytes"
	"strconv"
)

// GetDevInitCfgRequestToStringBuf .
func GetDevInitCfgRequestToStringBuf(r *GetInitCfgRequest) string {
	if r == nil {
		return "{}"
	}

	var buffer bytes.Buffer

	buffer.WriteString(`{"time" : "`)
	buffer.WriteString(strconv.FormatInt(r.Time, 10))
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_mac" : "`)
	buffer.WriteString(r.Mac)
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_type" : "`)
	buffer.WriteString(r.Type)
	buffer.WriteString(`"}`)

	return buffer.String()
}

// SaveDevDataRequestToStringBuf .
func SaveDevDataRequestToStringBuf(r *SaveDataRequest) string {
	if r == nil {
		return ""
	}

	var buffer bytes.Buffer

	buffer.WriteString(`{"time" : "`)
	buffer.WriteString(strconv.FormatInt(r.Time, 10))
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_meta" : "`)
	buffer.WriteString(r.Mac)
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_type" : "`)
	buffer.WriteString(r.Type)
	buffer.WriteString(`"}`)

	return buffer.String()
}

/* for slice:
buffer.WriteString(`[`)
for i, a := range slice {
	if i != 0 {
		buffer.WriteString(`,`)
	}

	...
}
buffer.WriteString(`]`)
*/
