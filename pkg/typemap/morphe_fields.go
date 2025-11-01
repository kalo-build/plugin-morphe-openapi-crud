package typemap

import (
	"fmt"

	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// MorpheFieldTypeToJSONSchema converts a Morphe field type to an OpenAPI Schema
func MorpheFieldTypeToJSONSchema(fieldType yaml.ModelFieldType) (*formatdef.Schema, error) {
	switch fieldType {
	case yaml.ModelFieldTypeAutoIncrement:
		return &formatdef.Schema{
			Type:     "integer",
			ReadOnly: true,
		}, nil

	case yaml.ModelFieldTypeBoolean:
		return &formatdef.Schema{
			Type: "boolean",
		}, nil

	case yaml.ModelFieldTypeDate:
		return &formatdef.Schema{
			Type:   "string",
			Format: "date",
		}, nil

	case yaml.ModelFieldTypeFloat:
		return &formatdef.Schema{
			Type:   "number",
			Format: "double",
		}, nil

	case yaml.ModelFieldTypeInteger:
		return &formatdef.Schema{
			Type: "integer",
		}, nil

	case yaml.ModelFieldTypeProtected:
		return &formatdef.Schema{
			Type:      "string",
			WriteOnly: true,
		}, nil

	case yaml.ModelFieldTypeSealed:
		return &formatdef.Schema{
			Type:      "string",
			WriteOnly: true,
		}, nil

	case yaml.ModelFieldTypeString:
		return &formatdef.Schema{
			Type: "string",
		}, nil

	case yaml.ModelFieldTypeTime:
		return &formatdef.Schema{
			Type:   "string",
			Format: "date-time",
		}, nil

	case yaml.ModelFieldTypeUUID:
		return &formatdef.Schema{
			Type:   "string",
			Format: "uuid",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported morphe field type for OpenAPI conversion: '%s'", fieldType)
	}
}

// StructureFieldTypeToJSONSchema converts a Morphe structure field type to an OpenAPI Schema
func StructureFieldTypeToJSONSchema(fieldType yaml.StructureFieldType) (*formatdef.Schema, error) {
	switch fieldType {
	case yaml.StructureFieldTypeBoolean:
		return &formatdef.Schema{
			Type: "boolean",
		}, nil

	case yaml.StructureFieldTypeDate:
		return &formatdef.Schema{
			Type:   "string",
			Format: "date",
		}, nil

	case yaml.StructureFieldTypeFloat:
		return &formatdef.Schema{
			Type:   "number",
			Format: "double",
		}, nil

	case yaml.StructureFieldTypeInteger:
		return &formatdef.Schema{
			Type: "integer",
		}, nil

	case yaml.StructureFieldTypeString:
		return &formatdef.Schema{
			Type: "string",
		}, nil

	case yaml.StructureFieldTypeTime:
		return &formatdef.Schema{
			Type:   "string",
			Format: "date-time",
		}, nil

	case yaml.StructureFieldTypeUUID:
		return &formatdef.Schema{
			Type:   "string",
			Format: "uuid",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported morphe structure field type for OpenAPI conversion: '%s'", fieldType)
	}
}

// GetPrimaryKeyType returns the JSON Schema type for a primary key field
func GetPrimaryKeyType(model yaml.Model) string {
	if len(model.Identifiers) == 0 {
		return "string"
	}

	// Get the primary identifier
	primaryID, hasPrimary := model.Identifiers["primary"]
	if !hasPrimary || len(primaryID.Fields) == 0 {
		return "string"
	}

	// Get the first field in the primary key
	fieldName := primaryID.Fields[0]
	field, hasField := model.Fields[fieldName]
	if !hasField {
		return "string"
	}

	// Map the field type to JSON Schema type
	switch field.Type {
	case yaml.ModelFieldTypeAutoIncrement:
		return "integer"
	case yaml.ModelFieldTypeInteger:
		return "integer"
	case yaml.ModelFieldTypeUUID:
		return "string"
	default:
		return "string"
	}
}
