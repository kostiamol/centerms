package proto

import (
	"bytes"
	"strconv"
)

// SetDevInitCfgRequestToStringBuf .
func SetDevInitCfgRequestToStringBuf(r *SetDevInitCfgRequest) string {
	if r == nil {
		return ""
	}

	var buffer bytes.Buffer

	buffer.WriteString(`{"time" : "`)
	buffer.WriteString(strconv.FormatInt(r.Time, 10))
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_meta" : "`)
	buffer.WriteString(devMetaToStringBuf(r.Meta))
	buffer.WriteString(`"}`)

	return buffer.String()
}

func devMetaToStringBuf(m *DevMeta) string {
	if m == nil {
		return "{}"
	}

	var buffer bytes.Buffer

	buffer.WriteString(`{"type" : "`)
	buffer.WriteString(m.Type)
	buffer.WriteString(`",`)

	buffer.WriteString(`"name" : "`)
	buffer.WriteString(m.Name)
	buffer.WriteString(`",`)

	buffer.WriteString(`"mac" : "`)
	buffer.WriteString(m.Mac)
	buffer.WriteString(`"}`)

	return buffer.String()
}

// SaveDevDataRequestToStringBuf .
func SaveDevDataRequestToStringBuf(r *SaveDevDataRequest) string {
	if r == nil {
		return ""
	}

	var buffer bytes.Buffer

	buffer.WriteString(`{"time" : "`)
	buffer.WriteString(strconv.FormatInt(r.Time, 10))
	buffer.WriteString(`",`)

	buffer.WriteString(`"dev_meta" : "`)
	buffer.WriteString(devMetaToStringBuf(r.Meta))
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
