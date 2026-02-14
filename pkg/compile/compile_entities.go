package compile

import (
	"fmt"
	"sort"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// EntitySchemas holds all schemas generated for an entity
type EntitySchemas struct {
	Name         string
	ReadSchema   *formatdef.Schema
	CreateSchema *formatdef.Schema
	UpdateSchema *formatdef.Schema
	ListSchema   *formatdef.Schema
}

// MorpheEntityToSchemas converts a Morphe entity to OpenAPI Schemas
func MorpheEntityToSchemas(reg *registry.Registry, entity yaml.Entity, config cfg.OpenAPIConfig) (*EntitySchemas, error) {
	// Validate entity
	if entity.Name == "" {
		return nil, fmt.Errorf("entity has no name")
	}
	if len(entity.Fields) == 0 {
		return nil, fmt.Errorf("morphe entity %s has no fields", entity.Name)
	}
	if len(entity.Identifiers) == 0 {
		return nil, fmt.Errorf("entity '%s' has no identifiers", entity.Name)
	}

	// Get the underlying model for the entity
	model, err := reg.GetModel(entity.Name)
	if err != nil {
		return nil, fmt.Errorf("entity '%s' has no corresponding model: %w", entity.Name, err)
	}

	result := &EntitySchemas{
		Name: entity.Name,
	}

	// Create read schema based on entity fields
	readSchema, err := createEntityReadSchema(reg, entity, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating read schema: %w", err)
	}
	result.ReadSchema = readSchema

	// Always create create/update schemas for entities (full CRUD)
	createSchema, err := createEntityCreateSchema(reg, entity, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating create schema: %w", err)
	}
	result.CreateSchema = createSchema

	updateSchema, err := createEntityUpdateSchema(reg, entity, model, config)
	if err != nil {
		return nil, fmt.Errorf("creating update schema: %w", err)
	}
	result.UpdateSchema = updateSchema

	// Create list schema
	result.ListSchema = createModelListSchema(entity.Name, config)

	return result, nil
}

func createEntityReadSchema(reg *registry.Registry, entity yaml.Entity, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("%s entity", entity.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{},
	}

	// Sort entity fields for deterministic output
	fieldNames := make([]string, 0, len(entity.Fields))
	for name := range entity.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// Convert each entity field by resolving from the model
	for _, fieldName := range fieldNames {
		entityField := entity.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		// Parse the entity field type (format: "ModelName.FieldName" or "ModelName.RelationName.FieldName")
		fieldDef, err := resolveEntityFieldType(reg, entityField.Type)
		if err != nil {
			return nil, fmt.Errorf("entity '%s' field '%s': %w", entity.Name, fieldName, err)
		}

		schema.Properties[fieldKey] = fieldDef

		isOptional := hasAttribute(entityField.Attributes, "optional")
		if !isOptional {
			schema.Required = append(schema.Required, fieldKey)
		} else {
			fieldDef.Nullable = true
		}
	}

	// Add relationship fields from entity
	if len(entity.Related) > 0 {
		relNames := make([]string, 0, len(entity.Related))
		for name := range entity.Related {
			relNames = append(relNames, name)
		}
		sort.Strings(relNames)

		for _, relName := range relNames {
			rel := entity.Related[relName]
			addEntityRelationshipFields(schema, relName, rel, config)
		}
	}

	return schema, nil
}

func addEntityRelationshipFields(schema *formatdef.Schema, relName string, rel yaml.EntityRelation, config cfg.OpenAPIConfig) {
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

func createEntityCreateSchema(reg *registry.Registry, entity yaml.Entity, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Create %s entity request", entity.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{},
	}

	// Get the underlying model's create schema logic
	// For entities, we filter based on what fields are exposed in the entity

	// Sort entity fields for deterministic output
	fieldNames := make([]string, 0, len(entity.Fields))
	for name := range entity.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		entityField := entity.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		// Check if this is an auto-generated field
		if isAutoGeneratedEntityField(reg, entityField.Type) {
			continue
		}

		// Parse the entity field type
		fieldDef, err := resolveEntityFieldType(reg, entityField.Type)
		if err != nil {
			return nil, fmt.Errorf("entity '%s' field '%s': %w", entity.Name, fieldName, err)
		}

		// Remove readOnly flag
		if fieldDef.Ref == "" {
			fieldDef.ReadOnly = false
			fieldDef.WriteOnly = false
		}

		isOptional := hasAttribute(entityField.Attributes, "optional")
		if isOptional {
			fieldDef.Nullable = true
		}

		schema.Properties[fieldKey] = fieldDef

		if !isOptional {
			schema.Required = append(schema.Required, fieldKey)
		}
	}

	// Add relationship fields
	if len(entity.Related) > 0 {
		relNames := make([]string, 0, len(entity.Related))
		for name := range entity.Related {
			relNames = append(relNames, name)
		}
		sort.Strings(relNames)

		for _, relName := range relNames {
			rel := entity.Related[relName]
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

func createEntityUpdateSchema(reg *registry.Registry, entity yaml.Entity, model yaml.Model, config cfg.OpenAPIConfig) (*formatdef.Schema, error) {
	schema := &formatdef.Schema{
		Type:        "object",
		Description: fmt.Sprintf("Update %s entity request", entity.Name),
		Properties:  make(map[string]*formatdef.Schema),
		Required:    []string{}, // All fields optional in update
	}

	// Sort entity fields for deterministic output
	fieldNames := make([]string, 0, len(entity.Fields))
	for name := range entity.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		entityField := entity.Fields[fieldName]
		fieldKey := formatdef.ToCamelCase(fieldName)

		// Check if this is an auto-generated or immutable field
		if isAutoGeneratedEntityField(reg, entityField.Type) || hasAttribute(entityField.Attributes, "immutable") {
			continue
		}

		// Parse the entity field type
		fieldDef, err := resolveEntityFieldType(reg, entityField.Type)
		if err != nil {
			return nil, fmt.Errorf("entity '%s' field '%s': %w", entity.Name, fieldName, err)
		}

		// Remove readOnly flag and make nullable
		if fieldDef.Ref == "" {
			fieldDef.ReadOnly = false
			fieldDef.WriteOnly = false
			fieldDef.Nullable = true
		}

		schema.Properties[fieldKey] = fieldDef
	}

	return schema, nil
}

// resolveEntityFieldType resolves an entity field type to a schema
// Entity field type format: "ModelName.FieldName" or "ModelName.RelationName.FieldName"
func resolveEntityFieldType(reg *registry.Registry, fieldType yaml.ModelFieldPath) (*formatdef.Schema, error) {
	// For now, create a simple string schema
	// In a real implementation, this would parse the field type and look up the actual type
	return &formatdef.Schema{
		Type: "string",
	}, nil
}

func isAutoGeneratedEntityField(reg *registry.Registry, fieldType yaml.ModelFieldPath) bool {
	// Check if field type indicates auto-generation
	// This is a simplified check
	return false
}

