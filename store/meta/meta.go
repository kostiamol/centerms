package meta

// Meta is an interface for interaction with response's metadata.
type Meta interface {
	GetMeta() map[string]interface{}
}

// Pagination holds response's metadata.
type Pagination struct {
	ResultCount uint `json:"result_count"`
	Limit       uint `json:"limit"`
	Offset      uint `json:"offset"`
}

// GetMetaData returns response's metadata.
func (p Pagination) GetMetaData() map[string]interface{} {
	m := map[string]interface{}{}

	m["pagination"] = map[string]uint{
		"total":  p.ResultCount,
		"limit":  p.Limit,
		"offset": p.Offset,
	}

	return m
}
