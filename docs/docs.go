// Package docs menyajikan spesifikasi OpenAPI/Swagger yang di-embed ke binary.
package docs

import (
	_ "embed"
	"encoding/json"
)

//go:embed api.json
var apiJSON []byte

// spec dapat diganti pada test untuk menguji penanganan kesalahan.
var spec = apiJSON

// GetSwaggerJSON mengembalikan spesifikasi OpenAPI dengan host dan skema dinamis sesuai pemanggil.
func GetSwaggerJSON(host, scheme string) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, err
	}
	doc["host"] = host
	doc["schemes"] = []string{scheme}
	return json.Marshal(doc)
}
