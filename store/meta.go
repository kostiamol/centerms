package store

// Meta .
type Meta interface {
	GetMeta() map[string]interface{}
}

// Pagination .
type Pagination struct {
	ResultCount uint `json:"result_count"`
	Limit       uint `json:"limit"`
	Offset      uint `json:"offset"`
}

// GetMeta return pagination meta
func (p Pagination) GetMetaData() map[string]interface{} {
	m := map[string]interface{}{}

	m["pagination"] = map[string]uint{
		"total":  p.ResultCount,
		"limit":  p.Limit,
		"offset": p.Offset,
	}

	return m
}
