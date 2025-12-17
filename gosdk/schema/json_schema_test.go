package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBooleanSchema(t *testing.T) {
	s := Boolean("A boolean flag")
	assert.Equal(t, TypeBoolean, s.Type)
	assert.Equal(t, "A boolean flag", s.Description)
}

func TestStringSchema(t *testing.T) {
	s := String("A string value")
	assert.Equal(t, TypeString, s.Type)
	assert.Equal(t, "A string value", s.Description)
}

func TestNumberSchema(t *testing.T) {
	s := Number("A numeric value")
	assert.Equal(t, TypeNumber, s.Type)
	assert.Equal(t, "A numeric value", s.Description)
}

func TestIntegerSchema(t *testing.T) {
	s := Integer("An integer value")
	assert.Equal(t, TypeInteger, s.Type)
	assert.Equal(t, "An integer value", s.Description)
}

func TestArraySchema(t *testing.T) {
	items := String("Array item")
	s := Array(items, "An array of strings")
	assert.Equal(t, TypeArray, s.Type)
	assert.Equal(t, "An array of strings", s.Description)
	assert.NotNil(t, s.Items)
	assert.Equal(t, TypeString, s.Items.Type)
}

func TestObjectSchema(t *testing.T) {
	props := map[string]*JSONSchema{
		"name": String("The name"),
		"age":  Number("The age"),
	}
	s := Object(props, []string{"name"})

	assert.Equal(t, TypeObject, s.Type)
	assert.Len(t, s.Properties, 2)
	assert.Equal(t, []string{"name"}, s.Required)
	assert.NotNil(t, s.AdditionalProperties)
	assert.False(t, s.AdditionalProperties.Allowed)
}

func TestJSONMarshal(t *testing.T) {
	props := map[string]*JSONSchema{
		"command": String("The command to execute"),
	}
	s := Object(props, []string{"command"})

	data, err := s.ToJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "object", result["type"])
	assert.Contains(t, result, "properties")
	assert.Equal(t, []interface{}{"command"}, result["required"])
}

func TestAdditionalPropertiesBoolean(t *testing.T) {
	ap := &AdditionalProperties{Allowed: false}

	data, err := json.Marshal(ap)
	require.NoError(t, err)
	assert.Equal(t, "false", string(data))

	ap.Allowed = true
	data, err = json.Marshal(ap)
	require.NoError(t, err)
	assert.Equal(t, "true", string(data))
}

func TestAdditionalPropertiesSchema(t *testing.T) {
	ap := &AdditionalProperties{
		Schema: String("Additional property"),
	}

	data, err := json.Marshal(ap)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, "string", result["type"])
}

func TestAdditionalPropertiesUnmarshal(t *testing.T) {
	// Test boolean
	ap := &AdditionalProperties{}
	err := json.Unmarshal([]byte("false"), ap)
	require.NoError(t, err)
	assert.False(t, ap.Allowed)
	assert.Nil(t, ap.Schema)

	// Test schema
	ap = &AdditionalProperties{}
	err = json.Unmarshal([]byte(`{"type":"string"}`), ap)
	require.NoError(t, err)
	assert.NotNil(t, ap.Schema)
	assert.Equal(t, TypeString, ap.Schema.Type)
}

func TestClone(t *testing.T) {
	original := Object(
		map[string]*JSONSchema{
			"name": String("Name"),
			"nested": Object(
				map[string]*JSONSchema{
					"value": Number("Value"),
				},
				[]string{"value"},
			),
		},
		[]string{"name"},
	)

	clone := original.Clone()

	// Verify clone is equal
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, len(original.Properties), len(clone.Properties))
	assert.Equal(t, original.Required, clone.Required)

	// Verify clone is independent
	clone.Properties["name"].Description = "Modified"
	assert.NotEqual(t, original.Properties["name"].Description, clone.Properties["name"].Description)
}

func TestSortedPropertyKeys(t *testing.T) {
	props := map[string]*JSONSchema{
		"zebra": String("Z"),
		"alpha": String("A"),
		"beta":  String("B"),
	}
	s := Object(props, nil)

	keys := s.SortedPropertyKeys()
	assert.Equal(t, []string{"alpha", "beta", "zebra"}, keys)
}

func TestNestedObjectSchema(t *testing.T) {
	innerProps := map[string]*JSONSchema{
		"id": Number("Identifier"),
	}
	inner := Object(innerProps, []string{"id"})

	outerProps := map[string]*JSONSchema{
		"data": inner,
	}
	outer := Object(outerProps, []string{"data"})

	data, err := outer.ToJSONPretty()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]interface{})
	dataSchema := props["data"].(map[string]interface{})
	assert.Equal(t, "object", dataSchema["type"])
}
