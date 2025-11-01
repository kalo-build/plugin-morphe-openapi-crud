package compile

import (
	"fmt"
	"sort"

	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// MorpheEnumToSchema converts a Morphe enum to an OpenAPI Schema
func MorpheEnumToSchema(enum yaml.Enum) (*formatdef.Schema, error) {
	// Validate enum
	if enum.Name == "" {
		return nil, yaml.ErrNoMorpheEnumName
	}
	if enum.Type == "" {
		return nil, yaml.ErrNoMorpheEnumType
	}
	if len(enum.Entries) == 0 {
		return nil, yaml.ErrNoMorpheEnumEntries
	}

	schema := &formatdef.Schema{
		Description: fmt.Sprintf("%s enumeration", enum.Name),
	}

	// Determine the JSON Schema type
	switch enum.Type {
	case yaml.EnumTypeString:
		schema.Type = "string"
	case yaml.EnumTypeInteger:
		schema.Type = "integer"
	case yaml.EnumTypeFloat:
		schema.Type = "number"
	default:
		return nil, fmt.Errorf("unsupported enum type: %s", enum.Type)
	}

	// Extract and sort enum values
	var enumValues []interface{}
	keys := make([]string, 0, len(enum.Entries))
	for k := range enum.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := enum.Entries[key]

		// Validate value type matches enum type
		switch enum.Type {
		case yaml.EnumTypeString:
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("enum entry '%s' value '%v' with type '%T' does not match the enum type of '%s'",
					key, value, value, enum.Type)
			}
		case yaml.EnumTypeInteger:
			if _, ok := value.(int); !ok {
				return nil, fmt.Errorf("enum entry '%s' value '%v' with type '%T' does not match the enum type of '%s'",
					key, value, value, enum.Type)
			}
		case yaml.EnumTypeFloat:
			if _, ok := value.(float64); !ok {
				return nil, fmt.Errorf("enum entry '%s' value '%v' with type '%T' does not match the enum type of '%s'",
					key, value, value, enum.Type)
			}
		}

		enumValues = append(enumValues, value)
	}

	schema.Enum = enumValues
	return schema, nil
}
