package compile

import (
	"fmt"
	"sort"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/typemap"
)

// MorpheStructureToSchema converts a Morphe structure to an OpenAPI Schema
func MorpheStructureToSchema(reg *registry.Registry, structure yaml.Structure) (*formatdef.Schema, error) {
	// Validate structure
	if structure.Name == "" {
		return nil, fmt.Errorf("structure has no name")
	}
	if len(structure.Fields) == 0 {
		return nil, fmt.Errorf("structure '%s' has no fields", structure.Name)
	}

	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("%s structure", structure.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{},
	}

	// Sort fields for deterministic output
	fieldNames := make([]string, 0, len(structure.Fields))
	for name := range structure.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// Convert each field
	for _, fieldName := range fieldNames {
		field := structure.Fields[fieldName]

		// Check if it's a reference to another structure
		if !yaml.IsStructureFieldTypePrimitive(field.Type) {
			// It's a reference to another structure
			structName := string(field.Type)
			schema.Properties[fieldName] = formatdef.SchemaRef(structName)
			schema.Required = append(schema.Required, fieldName)
			continue
		}

		// Convert primitive type
		fieldSchema, err := typemap.StructureFieldTypeToJSONSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("structure '%s' field '%s': %w", structure.Name, fieldName, err)
		}

		schema.Properties[fieldName] = fieldSchema
		schema.Required = append(schema.Required, fieldName)
	}

	return schema, nil
}
