package compile

import (
	"fmt"
	"sort"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/typemap"
)

// ModelSchemas holds all schemas generated for a model
type ModelSchemas struct {
	Name         string
	ReadSchema   *formatdef.Schema
	CreateSchema *formatdef.Schema
	UpdateSchema *formatdef.Schema
	ListSchema   *formatdef.Schema
}

// MorpheModelToSchemas converts a Morphe model to OpenAPI Schemas
func MorpheModelToSchemas(reg *registry.Registry, model yaml.Model, config cfg.OpenAPIConfig) (*ModelSchemas, error) {
	// Validate model
	if model.Name == "" {
		return nil, fmt.Errorf("morphe model has no name")
	}
	if len(model.Fields) == 0 {
		return nil, fmt.Errorf("morphe model has no fields")
	}
	if len(model.Identifiers) == 0 {
		return nil, fmt.Errorf("morphe model has no identifiers")
	}

	result := &ModelSchemas{
		Name: model.Name,
	}

	// Create read schema (full model with all fields)
	readSchema, err := createModelReadSchema(reg, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating read schema: %w", err)
	}
	result.ReadSchema = readSchema

	// Create create schema (without auto-generated fields)
	createSchema, err := createModelCreateSchema(reg, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating create schema: %w", err)
	}
	result.CreateSchema = createSchema

	// Create update schema (all fields optional except ID)
	updateSchema, err := createModelUpdateSchema(reg, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating update schema: %w", err)
	}
	result.UpdateSchema = updateSchema

	// Create list schema (paginated response)
	result.ListSchema = createModelListSchema(model.Name, config)

	return result, nil
}

func createModelReadSchema(reg *registry.Registry, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("%s model", model.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{},
	}

	// Sort fields for deterministic output
	fieldNames := make([]string, 0, len(model.Fields))
	for name := range model.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// Convert each field
	for _, fieldName := range fieldNames {
		field := model.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		isOptional := hasAttribute(field.Attributes, "optional")

		// Check if it's an enum reference
		if !yaml.IsModelFieldTypePrimitive(field.Type) {
			enumName := string(field.Type)
			enum, enumErr := reg.GetEnum(enumName)
			if enumErr == nil {
				// Reference to enum schema
				schema.Properties[fieldKey] = formatdef.SchemaRef(enum.Name)
				if !isOptional {
					schema.Required = append(schema.Required, fieldKey)
				}
				continue
			}
			return nil, fmt.Errorf("unsupported non-primitive type: %s", field.Type)
		}

		// Convert primitive type
		fieldSchema, err := typemap.MorpheFieldTypeToJSONSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", fieldName, err)
		}

		// Don't mark as required if it's optional or auto-generated
		if field.Type != yaml.ModelFieldTypeAutoIncrement && !isOptional {
			schema.Required = append(schema.Required, fieldKey)
		}

		if hasAttribute(field.Attributes, "immutable") {
			fieldSchema.ReadOnly = true
		}

		if isOptional {
			fieldSchema.Nullable = true
		}

		schema.Properties[fieldKey] = fieldSchema
	}

	// Add relationship fields
	if len(model.Related) > 0 {
		relNames := make([]string, 0, len(model.Related))
		for name := range model.Related {
			relNames = append(relNames, name)
		}
		sort.Strings(relNames)

		for _, relName := range relNames {
			rel := model.Related[relName]
			addRelationshipFields(schema, relName, rel, config)
		}
	}

	return schema, nil
}

func createModelCreateSchema(reg *registry.Registry, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Create %s request", model.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{},
	}

	// Sort fields for deterministic output
	fieldNames := make([]string, 0, len(model.Fields))
	for name := range model.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		field := model.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		// Skip auto-generated, immutable, protected, and sealed fields in create
		if field.Type == yaml.ModelFieldTypeAutoIncrement {
			continue
		}

		isImmutable := hasAttribute(field.Attributes, "immutable")
		isOptional := hasAttribute(field.Attributes, "optional")

		// Check if it's an enum reference
		if !yaml.IsModelFieldTypePrimitive(field.Type) {
			enumName := string(field.Type)
			enum, enumErr := reg.GetEnum(enumName)
			if enumErr == nil {
				schema.Properties[fieldKey] = formatdef.SchemaRef(enum.Name)
				if !isImmutable && !isOptional {
					schema.Required = append(schema.Required, fieldKey)
				}
				continue
			}
			return nil, fmt.Errorf("unsupported non-primitive type: %s", field.Type)
		}

		fieldSchema, err := typemap.MorpheFieldTypeToJSONSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", fieldName, err)
		}

		// Remove readOnly flag for create
		fieldSchema.ReadOnly = false
		fieldSchema.WriteOnly = false

		if isOptional {
			fieldSchema.Nullable = true
		}

		schema.Properties[fieldKey] = fieldSchema

		// Add to required unless it's immutable or optional
		if !isImmutable && !isOptional {
			schema.Required = append(schema.Required, fieldKey)
		}
	}

	// Add foreign key fields for ForOne/ForMany relationships
	if len(model.Related) > 0 {
		relNames := make([]string, 0, len(model.Related))
		for name := range model.Related {
			relNames = append(relNames, name)
		}
		sort.Strings(relNames)

		for _, relName := range relNames {
			rel := model.Related[relName]
			if rel.Type == "ForOne" {
				idFieldName := formatdef.ToCamelCase(relName) + "ID"
				schema.Properties[idFieldName] = &formatdef.Schema{
					Type:        "string",
					Description: fmt.Sprintf("Foreign key to %s", relName),
					Nullable:    true,
				}
			} else if rel.Type == "ForMany" {
				idsFieldName := formatdef.ToCamelCase(relName) + "IDs"
				schema.Properties[idsFieldName] = &formatdef.Schema{
					Type:        "array",
					Description: fmt.Sprintf("Foreign keys to %s", relName),
					Items: &formatdef.Schema{
						Type: "string",
					},
					Nullable: true,
				}
			} else if rel.Type == "ForOnePoly" {
				idFieldName := formatdef.ToCamelCase(relName) + "ID"
				typeFieldName := formatdef.ToCamelCase(relName) + "Type"
				schema.Properties[idFieldName] = &formatdef.Schema{
					Type:        "string",
					Description: fmt.Sprintf("Polymorphic ID for %s", relName),
					Nullable:    true,
				}
				schema.Properties[typeFieldName] = &formatdef.Schema{
					Type:        "string",
					Description: fmt.Sprintf("Polymorphic type for %s", relName),
					Nullable:    true,
				}
			} else if rel.Type == "ForManyPoly" {
				idsFieldName := formatdef.ToCamelCase(relName) + "IDs"
				typeFieldName := formatdef.ToCamelCase(relName) + "Type"
				schema.Properties[idsFieldName] = &formatdef.Schema{
					Type:        "array",
					Description: fmt.Sprintf("Polymorphic IDs for %s", relName),
					Items: &formatdef.Schema{
						Type: "string",
					},
					Nullable: true,
				}
				schema.Properties[typeFieldName] = &formatdef.Schema{
					Type:        "string",
					Description: fmt.Sprintf("Polymorphic type for %s", relName),
					Nullable:    true,
				}
			}
		}
	}

	return schema, nil
}

func createModelUpdateSchema(reg *registry.Registry, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Update %s request", model.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{}, // All fields are optional in update
	}

	// Sort fields for deterministic output
	fieldNames := make([]string, 0, len(model.Fields))
	for name := range model.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		field := model.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		// Skip auto-generated and immutable fields in update
		if field.Type == yaml.ModelFieldTypeAutoIncrement {
			continue
		}

		if hasAttribute(field.Attributes, "immutable") {
			continue
		}

		// Check if it's an enum reference
		if !yaml.IsModelFieldTypePrimitive(field.Type) {
			enumName := string(field.Type)
			enum, enumErr := reg.GetEnum(enumName)
			if enumErr == nil {
				schema.Properties[fieldKey] = formatdef.SchemaRef(enum.Name)
				continue
			}
			return nil, fmt.Errorf("unsupported non-primitive type: %s", field.Type)
		}

		fieldSchema, err := typemap.MorpheFieldTypeToJSONSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", fieldName, err)
		}

		// Remove readOnly flag for update
		fieldSchema.ReadOnly = false
		fieldSchema.WriteOnly = false
		fieldSchema.Nullable = true

		schema.Properties[fieldKey] = fieldSchema
	}

	// Add relationship fields (all optional)
	if len(model.Related) > 0 {
		relNames := make([]string, 0, len(model.Related))
		for name := range model.Related {
			relNames = append(relNames, name)
		}
		sort.Strings(relNames)

		for _, relName := range relNames {
			rel := model.Related[relName]
			if rel.Type == "ForOne" {
				idFieldName := formatdef.ToCamelCase(relName) + "ID"
				schema.Properties[idFieldName] = &formatdef.Schema{
					Type:        "string",
					Description: fmt.Sprintf("Foreign key to %s", relName),
					Nullable:    true,
				}
			} else if rel.Type == "ForMany" {
				idsFieldName := formatdef.ToCamelCase(relName) + "IDs"
				schema.Properties[idsFieldName] = &formatdef.Schema{
					Type:        "array",
					Description: fmt.Sprintf("Foreign keys to %s", relName),
					Items: &formatdef.Schema{
						Type: "string",
					},
					Nullable: true,
				}
			}
		}
	}

	return schema, nil
}

func createModelListSchema(modelName string, config cfg.OpenAPIConfig) *formatdef.Schema {
	return &formatdef.Schema{
		Type: "object",
		Properties: map[string]*formatdef.Schema{
			"data": {
				Type:  "array",
				Items: formatdef.SchemaRef(modelName),
			},
			"meta": {
				Type: "object",
				Properties: map[string]*formatdef.Schema{
					"page": {
						Type:        "integer",
						Description: "Current page number",
					},
					"pageSize": {
						Type:        "integer",
						Description: "Items per page",
					},
					"total": {
						Type:        "integer",
						Description: "Total number of items",
					},
					"totalPages": {
						Type:        "integer",
						Description: "Total number of pages",
					},
				},
				Required: []string{"page", "pageSize", "total", "totalPages"},
			},
		},
		Required: []string{"data", "meta"},
	}
}

// hasAttribute checks if the given attribute name is present in a slice of attributes
func hasAttribute(attrs []string, name string) bool {
	for _, attr := range attrs {
		if attr == name {
			return true
		}
	}
	return false
}

func addRelationshipFields(schema *formatdef.Schema, relName string, rel yaml.ModelRelation, config cfg.OpenAPIConfig) {
	switch rel.Type {
	case "ForOne":
		// Add ID field
		idFieldName := formatdef.ToCamelCase(relName) + "ID"
		schema.Properties[idFieldName] = &formatdef.Schema{
			Type:        "string",
			Description: fmt.Sprintf("Foreign key to %s", relName),
			Nullable:    true,
		}
		// Optionally add expanded object
		if config.Relations.Expand {
			objectFieldName := formatdef.ToCamelCase(relName)
			targetModel := relName
			if rel.Aliased != "" {
				targetModel = rel.Aliased
			}
			schema.Properties[objectFieldName] = &formatdef.Schema{
				Ref:         "#/components/schemas/" + targetModel,
				Description: fmt.Sprintf("Related %s object", targetModel),
				Nullable:    true,
			}
		}

	case "ForMany":
		// Add IDs array field
		idsFieldName := formatdef.ToCamelCase(relName) + "IDs"
		schema.Properties[idsFieldName] = &formatdef.Schema{
			Type:        "array",
			Description: fmt.Sprintf("Foreign keys to %s", relName),
			Items: &formatdef.Schema{
				Type: "string",
			},
			Nullable: true,
		}
		// Optionally add expanded objects
		if config.Relations.Expand {
			objectsFieldName := formatdef.ToCamelCase(relName) + "s"
			targetModel := relName
			if rel.Aliased != "" {
				targetModel = rel.Aliased
			}
			schema.Properties[objectsFieldName] = &formatdef.Schema{
				Type:        "array",
				Description: fmt.Sprintf("Related %s objects", targetModel),
				Items:       formatdef.SchemaRef(targetModel),
				Nullable:    true,
			}
		}

	case "ForOnePoly":
		// Add polymorphic ID and type fields
		idFieldName := formatdef.ToCamelCase(relName) + "ID"
		typeFieldName := formatdef.ToCamelCase(relName) + "Type"
		schema.Properties[idFieldName] = &formatdef.Schema{
			Type:        "string",
			Description: fmt.Sprintf("Polymorphic ID for %s", relName),
			Nullable:    true,
		}
		schema.Properties[typeFieldName] = &formatdef.Schema{
			Type:        "string",
			Description: fmt.Sprintf("Polymorphic type for %s", relName),
			Nullable:    true,
		}
		// Optionally add expanded union
		if config.Relations.Expand && len(rel.For) > 0 {
			objectFieldName := formatdef.ToCamelCase(relName)
			oneOfSchemas := make([]*formatdef.Schema, len(rel.For))
			for i, forModel := range rel.For {
				oneOfSchemas[i] = formatdef.SchemaRef(forModel)
			}
			schema.Properties[objectFieldName] = &formatdef.Schema{
				OneOf:       oneOfSchemas,
				Description: fmt.Sprintf("Related %s object (polymorphic)", relName),
				Nullable:    true,
			}
		}

	case "ForManyPoly":
		// Add polymorphic IDs and type fields
		idsFieldName := formatdef.ToCamelCase(relName) + "IDs"
		typeFieldName := formatdef.ToCamelCase(relName) + "Type"
		schema.Properties[idsFieldName] = &formatdef.Schema{
			Type:        "array",
			Description: fmt.Sprintf("Polymorphic IDs for %s", relName),
			Items: &formatdef.Schema{
				Type: "string",
			},
			Nullable: true,
		}
		schema.Properties[typeFieldName] = &formatdef.Schema{
			Type:        "string",
			Description: fmt.Sprintf("Polymorphic type for %s", relName),
			Nullable:    true,
		}

	case "HasOne", "HasMany", "HasOnePoly", "HasManyPoly":
		// These are inverse relationships, typically not included in schemas
		// but can be optionally expanded
		if config.Relations.Expand {
			targetModel := relName
			if rel.Aliased != "" {
				targetModel = rel.Aliased
			}
			if rel.Type == "HasOne" || rel.Type == "HasOnePoly" {
				objectFieldName := formatdef.ToCamelCase(relName)
				schema.Properties[objectFieldName] = &formatdef.Schema{
					Ref:         "#/components/schemas/" + targetModel,
					Description: fmt.Sprintf("Related %s object", targetModel),
					Nullable:    true,
				}
			} else {
				objectsFieldName := formatdef.ToCamelCase(relName) + "s"
				schema.Properties[objectsFieldName] = &formatdef.Schema{
					Type:        "array",
					Description: fmt.Sprintf("Related %s objects", targetModel),
					Items:       formatdef.SchemaRef(targetModel),
					Nullable:    true,
				}
			}
		}
	}
}
