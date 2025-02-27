package structs

import "net/http"

type StringMap struct {
	Entries []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"entries"`
}

func (m StringMap) StdMap() map[string]string {
	res := make(map[string]string, len(m.Entries))
	for _, entry := range m.Entries {
		res[entry.Key] = entry.Value
	}
	return res
}
func (m StringMap) HTTPHeaders() http.Header {
	res := make(http.Header, len(m.Entries))
	for _, entry := range m.Entries {
		res.Set(entry.Key, entry.Value)
	}
	return res
}
