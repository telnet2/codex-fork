// Package schema provides JSON Schema types for tool parameter definitions.
// This mirrors the Rust implementation in codex-rs/core/src/tools/spec.rs
package schema

import (
	"encoding/json"
	"sort"
)

// SchemaType represents the type of a JSON Schema
type SchemaType string

const (
	TypeBoolean SchemaType = "boolean"
	TypeString  SchemaType = "string"
	TypeNumber  SchemaType = "number"
	TypeInteger SchemaType = "integer"
	TypeArray   SchemaType = "array"
	TypeObject  SchemaType = "object"
)

// JSONSchema represents a JSON Schema definition for tool parameters.
// This is a subset of JSON Schema that covers the types needed for tool definitions.
type JSONSchema struct {
	Type                 SchemaType             `json:"type"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties *AdditionalProperties  `json:"additionalProperties,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
}

// AdditionalProperties represents either a boolean or a schema for additional properties
type AdditionalProperties struct {
	Allowed bool
	Schema  *JSONSchema
}

// MarshalJSON implements json.Marshaler for AdditionalProperties
func (ap AdditionalProperties) MarshalJSON() ([]byte, error) {
	if ap.Schema != nil {
		return json.Marshal(ap.Schema)
	}
	return json.Marshal(ap.Allowed)
}

// UnmarshalJSON implements json.Unmarshaler for AdditionalProperties
func (ap *AdditionalProperties) UnmarshalJSON(data []byte) error {
	// Try bool first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		ap.Allowed = b
		ap.Schema = nil
		return nil
	}

	// Try schema
	var schema JSONSchema
	if err := json.Unmarshal(data, &schema); err == nil {
		ap.Schema = &schema
		ap.Allowed = false
		return nil
	}

	return nil
}

// Boolean creates a boolean schema
func Boolean(description string) *JSONSchema {
	return &JSONSchema{
		Type:        TypeBoolean,
		Description: description,
	}
}

// String creates a string schema
func String(description string) *JSONSchema {
	return &JSONSchema{
		Type:        TypeString,
		Description: description,
	}
}

// Number creates a number schema
func Number(description string) *JSONSchema {
	return &JSONSchema{
		Type:        TypeNumber,
		Description: description,
	}
}

// Integer creates an integer schema
func Integer(description string) *JSONSchema {
	return &JSONSchema{
		Type:        TypeInteger,
		Description: description,
	}
}

// Array creates an array schema with the given item schema
func Array(items *JSONSchema, description string) *JSONSchema {
	return &JSONSchema{
		Type:        TypeArray,
		Items:       items,
		Description: description,
	}
}

// Object creates an object schema with the given properties
func Object(properties map[string]*JSONSchema, required []string) *JSONSchema {
	return &JSONSchema{
		Type:                 TypeObject,
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &AdditionalProperties{Allowed: false},
	}
}

// ObjectWithAdditional creates an object schema with additional properties configuration
func ObjectWithAdditional(properties map[string]*JSONSchema, required []string, additional *AdditionalProperties) *JSONSchema {
	return &JSONSchema{
		Type:                 TypeObject,
		Properties:           properties,
		Required:             required,
		AdditionalProperties: additional,
	}
}

// ToJSON converts the schema to a JSON representation with sorted keys for deterministic output
func (s *JSONSchema) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// ToJSONPretty converts the schema to a pretty-printed JSON representation
func (s *JSONSchema) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Clone creates a deep copy of the schema
func (s *JSONSchema) Clone() *JSONSchema {
	if s == nil {
		return nil
	}

	clone := &JSONSchema{
		Type:        s.Type,
		Description: s.Description,
	}

	if s.Properties != nil {
		clone.Properties = make(map[string]*JSONSchema, len(s.Properties))
		for k, v := range s.Properties {
			clone.Properties[k] = v.Clone()
		}
	}

	if s.Required != nil {
		clone.Required = make([]string, len(s.Required))
		copy(clone.Required, s.Required)
	}

	if s.AdditionalProperties != nil {
		clone.AdditionalProperties = &AdditionalProperties{
			Allowed: s.AdditionalProperties.Allowed,
			Schema:  s.AdditionalProperties.Schema.Clone(),
		}
	}

	if s.Items != nil {
		clone.Items = s.Items.Clone()
	}

	return clone
}

// SortedPropertyKeys returns the property keys in sorted order
func (s *JSONSchema) SortedPropertyKeys() []string {
	if s.Properties == nil {
		return nil
	}

	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
