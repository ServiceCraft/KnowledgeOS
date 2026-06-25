package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// jsonSchema is the subset of JSON Schema understood by the tool framework. It
// is intentionally small: object/scalar/array typing, required fields, string
// enums and numeric/length bounds cover every tool argument we expose, without
// pulling in a heavyweight schema dependency.
type jsonSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]*jsonSchema `json:"properties"`
	Required   []string               `json:"required"`
	Items      *jsonSchema            `json:"items"`
	Enum       []interface{}          `json:"enum"`
	Minimum    *float64               `json:"minimum"`
	Maximum    *float64               `json:"maximum"`
	MinLength  *int                   `json:"minLength"`
}

// validateArguments checks raw tool-call arguments against a tool's JSON schema.
// It returns a *ValidationError describing the first mismatch, or nil when the
// arguments satisfy the schema.
func validateArguments(schemaRaw, args json.RawMessage) error {
	if len(schemaRaw) == 0 {
		return nil
	}
	var schema jsonSchema
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		return &ValidationError{Msg: "invalid tool schema"}
	}

	raw := args
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value interface{}
	if err := dec.Decode(&value); err != nil {
		return &ValidationError{Msg: "arguments are not valid JSON"}
	}
	if err := schema.validate("arguments", value); err != nil {
		return &ValidationError{Msg: err.Error()}
	}
	return nil
}

func (s *jsonSchema) validate(path string, value interface{}) error {
	switch s.Type {
	case "object":
		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		for _, req := range s.Required {
			if _, present := obj[req]; !present {
				return fmt.Errorf("%s.%s is required", path, req)
			}
		}
		for name, prop := range s.Properties {
			if prop == nil {
				continue
			}
			if child, present := obj[name]; present {
				if err := prop.validate(path+"."+name, child); err != nil {
					return err
				}
			}
		}
	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("%s must be an array", path)
		}
		if s.Items != nil {
			for i, item := range arr {
				if err := s.Items.validate(path+"["+strconv.Itoa(i)+"]", item); err != nil {
					return err
				}
			}
		}
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a string", path)
		}
		if s.MinLength != nil && len([]rune(str)) < *s.MinLength {
			return fmt.Errorf("%s must be at least %d characters", path, *s.MinLength)
		}
		if err := s.checkEnum(path, str); err != nil {
			return err
		}
	case "integer":
		n, ok := toFloat(value)
		if !ok {
			return fmt.Errorf("%s must be an integer", path)
		}
		if n != math.Trunc(n) {
			return fmt.Errorf("%s must be an integer", path)
		}
		if err := s.checkBounds(path, n); err != nil {
			return err
		}
	case "number":
		n, ok := toFloat(value)
		if !ok {
			return fmt.Errorf("%s must be a number", path)
		}
		if err := s.checkBounds(path, n); err != nil {
			return err
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	case "":
		// Untyped node: no structural constraint to enforce.
	default:
		// Unknown type keyword: be lenient rather than reject valid input.
	}
	return nil
}

func (s *jsonSchema) checkBounds(path string, n float64) error {
	if s.Minimum != nil && n < *s.Minimum {
		return fmt.Errorf("%s must be >= %v", path, *s.Minimum)
	}
	if s.Maximum != nil && n > *s.Maximum {
		return fmt.Errorf("%s must be <= %v", path, *s.Maximum)
	}
	return nil
}

func (s *jsonSchema) checkEnum(path, value string) error {
	if len(s.Enum) == 0 {
		return nil
	}
	for _, candidate := range s.Enum {
		if fmt.Sprint(candidate) == value {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of the allowed values", path)
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return v, true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}
