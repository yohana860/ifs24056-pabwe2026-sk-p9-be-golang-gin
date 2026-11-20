package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSwaggerJSON(t *testing.T) {
	b, err := GetSwaggerJSON("localhost:9999", "https")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(b, &doc))
	assert.Equal(t, "localhost:9999", doc["host"])
	assert.Equal(t, []any{"https"}, doc["schemes"])
	assert.NotEmpty(t, doc["paths"])
}

func TestGetSwaggerJSONInvalidSpec(t *testing.T) {
	old := spec
	spec = []byte("{rusak")
	defer func() { spec = old }()
	_, err := GetSwaggerJSON("h", "http")
	assert.Error(t, err)
}
