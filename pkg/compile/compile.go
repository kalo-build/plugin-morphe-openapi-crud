package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/kalo-build/morphe-go/pkg/registry"
	rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// MorpheCompileConfig holds configuration for the compilation process
type MorpheCompileConfig struct {
	MorpheLoadRegistryConfig rcfg.MorpheLoadRegistryConfig
	OpenAPIConfig            cfg.OpenAPIConfig
	OutputPath               string
}

// MorpheToOpenAPI compiles a Morphe registry to an OpenAPI 3.1 document
func MorpheToOpenAPI(config MorpheCompileConfig) error {
	// Load the Morphe registry
	reg, err := registry.LoadMorpheRegistry(registry.LoadMorpheRegistryHooks{}, config.MorpheLoadRegistryConfig)
	if err != nil {
		return fmt.Errorf("loading morphe registry: %w", err)
	}

	// Create OpenAPI document
	doc := formatdef.NewOpenAPIDocument()

	// Set basic info
	doc.Info.Title = "Generated API"
	doc.Info.Description = "API generated from Morphe schema"
	doc.Info.Version = "1.0.0"

	// Add servers
	if len(config.OpenAPIConfig.Servers) > 0 {
		doc.Servers = make([]formatdef.Server, len(config.OpenAPIConfig.Servers))
		for i, srv := range config.OpenAPIConfig.Servers {
			doc.Servers[i] = formatdef.Server{
				URL:         srv.URL,
				Description: srv.Description,
			}
		}
	}

	// Add security schemes
	if err := addSecuritySchemes(doc, config.OpenAPIConfig); err != nil {
		return fmt.Errorf("adding security schemes: %w", err)
	}

	// Initialize reference tracker
	refTracker := NewReferenceTracker()

	// Track references based on resource source
	resourceSource := config.OpenAPIConfig.ResourceSource
	if resourceSource == "" {
		resourceSource = "entities" // Default
	}

	if resourceSource == "models" || resourceSource == "both" {
		// Track references from models
		allModels := reg.GetAllModels()
		for _, model := range allModels {
			refTracker.TrackModelReferences(reg, model)
		}
	}

	if resourceSource == "entities" || resourceSource == "both" {
		// Track references from entities
		allEntities := reg.GetAllEntities()
		for _, entity := range allEntities {
			refTracker.TrackEntityReferences(reg, entity)
		}
	}

	// Process enums (with filtering)
	if err := processEnums(doc, reg, refTracker, config.OpenAPIConfig); err != nil {
		return fmt.Errorf("processing enums: %w", err)
	}

	// Process structures (with filtering)
	if err := processStructures(doc, reg, refTracker, config.OpenAPIConfig); err != nil {
		return fmt.Errorf("processing structures: %w", err)
	}

	// Process models
	if resourceSource == "models" || resourceSource == "both" {
		if err := processModels(doc, reg, config.OpenAPIConfig); err != nil {
			return fmt.Errorf("processing models: %w", err)
		}
	}

	// Process entities
	if resourceSource == "entities" || resourceSource == "both" {
		if err := processEntities(doc, reg, config.OpenAPIConfig); err != nil {
			return fmt.Errorf("processing entities: %w", err)
		}
	}

	// Add common components
	addCommonComponents(doc, config.OpenAPIConfig)

	// Sort tags
	sortTags(doc)

	// Write output (segmented or monolithic)
	if config.OpenAPIConfig.SegmentedOutput {
		if err := writeSegmentedOutput(doc, config, refTracker); err != nil {
			return fmt.Errorf("writing segmented output: %w", err)
		}
	} else {
		if err := writeOpenAPIDocument(doc, config); err != nil {
			return fmt.Errorf("writing openapi document: %w", err)
		}
	}

	return nil
}

func addSecuritySchemes(doc *formatdef.OpenAPIDocument, config cfg.OpenAPIConfig) error {
	switch config.Auth.Scheme {
	case "bearer":
		doc.Components.SecuritySchemes["bearerAuth"] = formatdef.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Bearer authentication with JWT token",
		}
		doc.Security = []formatdef.SecurityRequirement{
			{"bearerAuth": []string{}},
		}

	case "oauth2":
		if config.Auth.OAuth2Flows == nil {
			return fmt.Errorf("oauth2 scheme requires oauth2Flows configuration")
		}

		flows := &formatdef.OAuthFlows{}
		if config.Auth.OAuth2Flows.AuthorizationCode != nil {
			flows.AuthorizationCode = &formatdef.OAuthFlow{
				AuthorizationURL: config.Auth.OAuth2Flows.AuthorizationCode.AuthorizationURL,
				TokenURL:         config.Auth.OAuth2Flows.AuthorizationCode.TokenURL,
				RefreshURL:       config.Auth.OAuth2Flows.AuthorizationCode.RefreshURL,
				Scopes:           config.Auth.OAuth2Flows.AuthorizationCode.Scopes,
			}
		}
		if config.Auth.OAuth2Flows.ClientCredentials != nil {
			flows.ClientCredentials = &formatdef.OAuthFlow{
				TokenURL:   config.Auth.OAuth2Flows.ClientCredentials.TokenURL,
				RefreshURL: config.Auth.OAuth2Flows.ClientCredentials.RefreshURL,
				Scopes:     config.Auth.OAuth2Flows.ClientCredentials.Scopes,
			}
		}

		doc.Components.SecuritySchemes["oauth2"] = formatdef.SecurityScheme{
			Type:        "oauth2",
			Description: "OAuth2 authentication",
			Flows:       flows,
		}
		doc.Security = []formatdef.SecurityRequirement{
			{"oauth2": []string{}},
		}
	}

	return nil
}

func processEnums(doc *formatdef.OpenAPIDocument, reg *registry.Registry, refTracker *ReferenceTracker, config cfg.OpenAPIConfig) error {
	allEnums := reg.GetAllEnums()

	// Get sorted enum names for deterministic output
	enumNames := make([]string, 0, len(allEnums))
	for enumName := range allEnums {
		enumNames = append(enumNames, enumName)
	}
	sort.Strings(enumNames)

	for _, enumName := range enumNames {
		// Skip unreferenced enums unless IncludeAllSchemas is true
		if !refTracker.ShouldIncludeEnum(enumName, config.IncludeAllSchemas) {
			continue
		}

		enum := allEnums[enumName]

		schema, err := MorpheEnumToSchema(enum)
		if err != nil {
			return fmt.Errorf("converting enum '%s': %w", enumName, err)
		}

		doc.Components.Schemas[enumName] = *schema
	}

	return nil
}

func processStructures(doc *formatdef.OpenAPIDocument, reg *registry.Registry, refTracker *ReferenceTracker, config cfg.OpenAPIConfig) error {
	allStructures := reg.GetAllStructures()

	// Get sorted structure names for deterministic output
	structureNames := make([]string, 0, len(allStructures))
	for structureName := range allStructures {
		structureNames = append(structureNames, structureName)
	}
	sort.Strings(structureNames)

	for _, structureName := range structureNames {
		// Skip unreferenced structures unless IncludeAllSchemas is true
		if !refTracker.ShouldIncludeStructure(structureName, config.IncludeAllSchemas) {
			continue
		}

		structure := allStructures[structureName]

		schema, err := MorpheStructureToSchema(reg, structure)
		if err != nil {
			return fmt.Errorf("converting structure '%s': %w", structureName, err)
		}

		// Track nested structure references
		refTracker.TrackStructureReferences(reg, structure)

		doc.Components.Schemas[structureName] = *schema
	}

	return nil
}

func processModels(doc *formatdef.OpenAPIDocument, reg *registry.Registry, config cfg.OpenAPIConfig) error {
	allModels := reg.GetAllModels()

	// Get sorted model names for deterministic output
	modelNames := make([]string, 0, len(allModels))
	for modelName := range allModels {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)

	for _, modelName := range modelNames {
		model := allModels[modelName]

		// Generate schemas
		schemas, err := MorpheModelToSchemas(reg, model, config)
		if err != nil {
			return fmt.Errorf("converting model '%s': %w", modelName, err)
		}

		// Add schemas to components
		doc.Components.Schemas[modelName] = *schemas.ReadSchema
		doc.Components.Schemas[modelName+"Create"] = *schemas.CreateSchema
		doc.Components.Schemas[modelName+"Update"] = *schemas.UpdateSchema
		doc.Components.Schemas[modelName+"List"] = *schemas.ListSchema

		// Generate CRUD paths
		if err := addModelPaths(doc, model, config); err != nil {
			return fmt.Errorf("adding paths for model '%s': %w", modelName, err)
		}

		// Add tag
		doc.Tags = append(doc.Tags, formatdef.Tag{
			Name:        modelName,
			Description: fmt.Sprintf("Operations on %s", modelName),
		})
	}

	return nil
}

func processEntities(doc *formatdef.OpenAPIDocument, reg *registry.Registry, config cfg.OpenAPIConfig) error {
	allEntities := reg.GetAllEntities()

	// Get sorted entity names for deterministic output
	entityNames := make([]string, 0, len(allEntities))
	for entityName := range allEntities {
		entityNames = append(entityNames, entityName)
	}
	sort.Strings(entityNames)

	for _, entityName := range entityNames {
		entity := allEntities[entityName]

		// Generate schemas
		schemas, err := MorpheEntityToSchemas(reg, entity, config)
		if err != nil {
			return fmt.Errorf("converting entity '%s': %w", entityName, err)
		}

		// Add schemas to components
		doc.Components.Schemas[entityName] = *schemas.ReadSchema
		if schemas.CreateSchema != nil {
			doc.Components.Schemas[entityName+"Create"] = *schemas.CreateSchema
		}
		if schemas.UpdateSchema != nil {
			doc.Components.Schemas[entityName+"Update"] = *schemas.UpdateSchema
		}
		doc.Components.Schemas[entityName+"List"] = *schemas.ListSchema

		// Generate paths based on exposure level
		if err := addEntityPaths(doc, reg, entity, config); err != nil {
			return fmt.Errorf("adding paths for entity '%s': %w", entityName, err)
		}

		// Add tag
		doc.Tags = append(doc.Tags, formatdef.Tag{
			Name:        entityName,
			Description: fmt.Sprintf("Operations on %s entity", entityName),
		})
	}

	return nil
}

func addModelPaths(doc *formatdef.OpenAPIDocument, model yaml.Model, config cfg.OpenAPIConfig) error {
	modelName := model.Name
	collectionName := formatdef.ConvertName(modelName, config.Naming, config.Collections.Pluralize)
	singleName := formatdef.ConvertName(modelName, config.Naming, false)

	// Determine ID type from model's primary key
	idType := getPrimaryKeyJSONType(model)

	basePath := formatdef.BuildPath(config.BasePath, collectionName)
	itemPath := formatdef.BuildPath(basePath, "{"+config.IdParam+"}")

	// Add reusable pagination parameters to components if not already there
	addPaginationParametersToComponents(doc, config)

	// GET /models - List
	listOp := &formatdef.Operation{
		Tags:        []string{modelName},
		Summary:     fmt.Sprintf("List %s", collectionName),
		Description: fmt.Sprintf("Retrieve a paginated list of %s", collectionName),
		OperationID: formatdef.CreateOperationID(modelName, "list"),
		Parameters:  formatdef.CreatePaginationParameterRefs(config.Pagination.Type),
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(modelName+"List"),
				fmt.Sprintf("List of %s", collectionName),
			),
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	// POST /models - Create
	createOp := &formatdef.Operation{
		Tags:        []string{modelName},
		Summary:     fmt.Sprintf("Create %s", singleName),
		Description: fmt.Sprintf("Create a new %s", singleName),
		OperationID: formatdef.CreateOperationID(modelName, "create"),
		RequestBody: &formatdef.RequestBody{
			Description: fmt.Sprintf("%s to create", modelName),
			Required:    true,
			Content: map[string]formatdef.MediaType{
				"application/json": {
					Schema: formatdef.SchemaRef(modelName + "Create"),
				},
			},
		},
		Responses: map[string]formatdef.Response{
			"201": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(modelName),
				fmt.Sprintf("Created %s", singleName),
			),
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	// GET /models/{id} - Read
	readOp := &formatdef.Operation{
		Tags:        []string{modelName},
		Summary:     fmt.Sprintf("Get %s", singleName),
		Description: fmt.Sprintf("Retrieve a single %s by ID", singleName),
		OperationID: formatdef.CreateOperationID(modelName, "get"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", modelName)),
		},
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(modelName),
				fmt.Sprintf("%s details", modelName),
			),
			"404": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	// PATCH /models/{id} - Update
	updateOp := &formatdef.Operation{
		Tags:        []string{modelName},
		Summary:     fmt.Sprintf("Update %s", singleName),
		Description: fmt.Sprintf("Update an existing %s", singleName),
		OperationID: formatdef.CreateOperationID(modelName, "update"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", modelName)),
		},
		RequestBody: &formatdef.RequestBody{
			Description: fmt.Sprintf("%s updates", modelName),
			Required:    true,
			Content: map[string]formatdef.MediaType{
				"application/json": {
					Schema: formatdef.SchemaRef(modelName + "Update"),
				},
			},
		},
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(modelName),
				fmt.Sprintf("Updated %s", singleName),
			),
			"404": {Ref: "#/components/responses/Error"},
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	// DELETE /models/{id} - Delete
	deleteOp := &formatdef.Operation{
		Tags:        []string{modelName},
		Summary:     fmt.Sprintf("Delete %s", singleName),
		Description: fmt.Sprintf("Delete a %s", singleName),
		OperationID: formatdef.CreateOperationID(modelName, "delete"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", modelName)),
		},
		Responses: map[string]formatdef.Response{
			"204": {
				Description: "Successfully deleted",
			},
			"404": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	// Add paths to document
	doc.Paths[basePath] = formatdef.PathItem{
		Get:  listOp,
		Post: createOp,
	}

	doc.Paths[itemPath] = formatdef.PathItem{
		Get:    readOp,
		Patch:  updateOp,
		Delete: deleteOp,
	}

	return nil
}

func addEntityPaths(doc *formatdef.OpenAPIDocument, reg *registry.Registry, entity yaml.Entity, config cfg.OpenAPIConfig) error {
	entityName := entity.Name
	collectionName := formatdef.ConvertName(entityName, config.Naming, config.Collections.Pluralize)
	singleName := formatdef.ConvertName(entityName, config.Naming, false)

	// Get the model to determine ID type
	model, err := reg.GetModel(entityName)
	idType := "string"
	if err == nil {
		idType = getPrimaryKeyJSONType(model)
	}

	basePath := formatdef.BuildPath(config.BasePath, collectionName)
	itemPath := formatdef.BuildPath(basePath, "{"+config.IdParam+"}")

	pathItem := formatdef.PathItem{}

	// Always add read operations
	listOp := &formatdef.Operation{
		Tags:        []string{entityName},
		Summary:     fmt.Sprintf("List %s", collectionName),
		Description: fmt.Sprintf("Retrieve a paginated list of %s", collectionName),
		OperationID: formatdef.CreateOperationID(entityName, "list"),
		Parameters:  formatdef.CreatePaginationParameterRefs(config.Pagination.Type),
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(entityName+"List"),
				fmt.Sprintf("List of %s", collectionName),
			),
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	readOp := &formatdef.Operation{
		Tags:        []string{entityName},
		Summary:     fmt.Sprintf("Get %s", singleName),
		Description: fmt.Sprintf("Retrieve a single %s by ID", singleName),
		OperationID: formatdef.CreateOperationID(entityName, "get"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", entityName)),
		},
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(entityName),
				fmt.Sprintf("%s details", entityName),
			),
			"404": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	pathItem.Get = listOp
	doc.Paths[itemPath] = formatdef.PathItem{Get: readOp}

	// Add write operations (entities always have full CRUD)
	createOp := &formatdef.Operation{
		Tags:        []string{entityName},
		Summary:     fmt.Sprintf("Create %s", singleName),
		Description: fmt.Sprintf("Create a new %s", singleName),
		OperationID: formatdef.CreateOperationID(entityName, "create"),
		RequestBody: &formatdef.RequestBody{
			Description: fmt.Sprintf("%s to create", entityName),
			Required:    true,
			Content: map[string]formatdef.MediaType{
				"application/json": {
					Schema: formatdef.SchemaRef(entityName + "Create"),
				},
			},
		},
		Responses: map[string]formatdef.Response{
			"201": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(entityName),
				fmt.Sprintf("Created %s", singleName),
			),
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	updateOp := &formatdef.Operation{
		Tags:        []string{entityName},
		Summary:     fmt.Sprintf("Update %s", singleName),
		Description: fmt.Sprintf("Update an existing %s", singleName),
		OperationID: formatdef.CreateOperationID(entityName, "update"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", entityName)),
		},
		RequestBody: &formatdef.RequestBody{
			Description: fmt.Sprintf("%s updates", entityName),
			Required:    true,
			Content: map[string]formatdef.MediaType{
				"application/json": {
					Schema: formatdef.SchemaRef(entityName + "Update"),
				},
			},
		},
		Responses: map[string]formatdef.Response{
			"200": formatdef.CreateSimpleResponse(
				formatdef.SchemaRef(entityName),
				fmt.Sprintf("Updated %s", singleName),
			),
			"404": {Ref: "#/components/responses/Error"},
			"400": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	deleteOp := &formatdef.Operation{
		Tags:        []string{entityName},
		Summary:     fmt.Sprintf("Delete %s", singleName),
		Description: fmt.Sprintf("Delete a %s", singleName),
		OperationID: formatdef.CreateOperationID(entityName, "delete"),
		Parameters: []formatdef.Parameter{
			formatdef.CreateIDParameter(config.IdParam, idType, fmt.Sprintf("%s ID", entityName)),
		},
		Responses: map[string]formatdef.Response{
			"204": {
				Description: "Successfully deleted",
			},
			"404": {Ref: "#/components/responses/Error"},
			"500": {Ref: "#/components/responses/Error"},
		},
	}

	pathItem.Post = createOp
	itemPathItem := doc.Paths[itemPath]
	itemPathItem.Patch = updateOp
	itemPathItem.Delete = deleteOp
	doc.Paths[itemPath] = itemPathItem

	doc.Paths[basePath] = pathItem

	return nil
}

func addCommonComponents(doc *formatdef.OpenAPIDocument, config cfg.OpenAPIConfig) {
	// Add common error response
	doc.Components.Responses["Error"] = formatdef.CreateErrorResponse("Error response")

	// Add common pagination parameters
	addPaginationParametersToComponents(doc, config)
}

func addPaginationParametersToComponents(doc *formatdef.OpenAPIDocument, config cfg.OpenAPIConfig) {
	if config.Pagination.Type == "cursor" {
		if _, exists := doc.Components.Parameters["cursor"]; !exists {
			doc.Components.Parameters["cursor"] = formatdef.Parameter{
				Name:        "cursor",
				In:          "query",
				Description: "Pagination cursor",
				Schema: &formatdef.Schema{
					Type: "string",
				},
			}
		}
		if _, exists := doc.Components.Parameters["limit"]; !exists {
			max := float64(config.Pagination.MaxPageSize)
			doc.Components.Parameters["limit"] = formatdef.Parameter{
				Name:        "limit",
				In:          "query",
				Description: "Number of items to return",
				Schema: &formatdef.Schema{
					Type:    "integer",
					Minimum: float64Ptr(1),
					Maximum: &max,
				},
			}
		}
	} else {
		// Page-based pagination
		if _, exists := doc.Components.Parameters["page"]; !exists {
			doc.Components.Parameters["page"] = formatdef.Parameter{
				Name:        "page",
				In:          "query",
				Description: "Page number",
				Schema: &formatdef.Schema{
					Type:    "integer",
					Minimum: float64Ptr(1),
				},
			}
		}
		if _, exists := doc.Components.Parameters["pageSize"]; !exists {
			max := float64(config.Pagination.MaxPageSize)
			doc.Components.Parameters["pageSize"] = formatdef.Parameter{
				Name:        "pageSize",
				In:          "query",
				Description: "Number of items per page",
				Schema: &formatdef.Schema{
					Type:    "integer",
					Minimum: float64Ptr(1),
					Maximum: &max,
				},
			}
		}
	}
}

func getPrimaryKeyJSONType(model yaml.Model) string {
	if len(model.Identifiers) == 0 {
		return "string"
	}

	primaryID, hasPrimary := model.Identifiers["primary"]
	if !hasPrimary || len(primaryID.Fields) == 0 {
		return "string"
	}

	fieldName := primaryID.Fields[0]
	field, hasField := model.Fields[fieldName]
	if !hasField {
		return "string"
	}

	switch field.Type {
	case yaml.ModelFieldTypeAutoIncrement, yaml.ModelFieldTypeInteger:
		return "integer"
	case yaml.ModelFieldTypeUUID:
		return "string"
	default:
		return "string"
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

func sortTags(doc *formatdef.OpenAPIDocument) {
	sort.Slice(doc.Tags, func(i, j int) bool {
		return doc.Tags[i].Name < doc.Tags[j].Name
	})
}

func writeOpenAPIDocument(doc *formatdef.OpenAPIDocument, config MorpheCompileConfig) error {
	var data []byte
	var err error

	if config.OpenAPIConfig.OutputFormat == "json" {
		data, err = json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling to JSON: %w", err)
		}
	} else {
		// Default to YAML
		data, err = yamlv3.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshaling to YAML: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(config.OutputPath, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func writeSegmentedOutput(doc *formatdef.OpenAPIDocument, config MorpheCompileConfig, refTracker *ReferenceTracker) error {
	writer := NewSegmentedWriter(config.OutputPath, config.OpenAPIConfig.OutputFormat)

	// Determine ID strategy for annotations
	resourceType := config.OpenAPIConfig.ResourceSource
	if resourceType == "" {
		resourceType = "models"
	}

	// Write enum fragments
	for enumName, schema := range doc.Components.Schemas {
		// Check if this is an enum (has Enum field populated)
		if schema.Enum != nil && len(schema.Enum) > 0 {
			annotations := NewEnumAnnotations(enumName)
			if err := writer.WriteEnumSchema(enumName, schema, annotations); err != nil {
				return fmt.Errorf("writing enum %s: %w", enumName, err)
			}
		}
	}

	// Write entity/model schemas and DTOs
	for schemaName, schema := range doc.Components.Schemas {
		if schema.Enum != nil {
			continue // Skip enums (already handled)
		}

		// Check if this is a DTO (ends with Create/Update/List)
		if isDTOSchema(schemaName) {
			dtoType := getDTOType(schemaName)
			baseName := getBaseName(schemaName, dtoType)
			annotations := NewDTOAnnotations(baseName, dtoType)
			if err := writer.WriteDTOSchema(baseName, dtoType, schema, annotations); err != nil {
				return fmt.Errorf("writing DTO %s: %w", schemaName, err)
			}
			continue
		}

		// Check if this is an entity/model (has Create/Update/List variants)
		if isEntityOrModelSchema(schemaName, doc.Components.Schemas) {
			idStrategy := getIDStrategy(schema)
			var annotations KaloMorpheAnnotations
			if resourceType == "entities" {
				annotations = NewEntityAnnotations(schemaName, idStrategy)
			} else {
				annotations = NewModelAnnotations(schemaName, idStrategy)
			}
			if err := writer.WriteEntitySchema(schemaName, schema, annotations); err != nil {
				return fmt.Errorf("writing schema %s: %w", schemaName, err)
			}
			continue
		}

		// Otherwise, it's a structure
		isReferenced := refTracker.Structures[schemaName]
		annotations := NewStructureAnnotations(schemaName, isReferenced)
		if err := writer.WriteStructureSchema(schemaName, schema, annotations); err != nil {
			return fmt.Errorf("writing structure %s: %w", schemaName, err)
		}
	}

	// Write paths fragments (grouped by resource)
	pathsByResource := groupPathsByResource(doc.Paths, config.OpenAPIConfig)
	for resourceName, paths := range pathsByResource {
		resourceType := "model" // or "entity" based on config
		annotations := NewPathAnnotations(resourceName, "crud", resourceType)
		if err := writer.WritePathsFragment(resourceName, paths, annotations); err != nil {
			return fmt.Errorf("writing paths for %s: %w", resourceName, err)
		}
	}

	// Write shared parameters
	if len(doc.Components.Parameters) > 0 {
		annotations := NewSharedComponentAnnotations("parameter")
		if err := writer.WriteParametersFragment("pagination", doc.Components.Parameters, annotations); err != nil {
			return fmt.Errorf("writing pagination parameters: %w", err)
		}
	}

	// Write shared responses
	if len(doc.Components.Responses) > 0 {
		annotations := NewSharedComponentAnnotations("response")
		if err := writer.WriteResponsesFragment("error", doc.Components.Responses, annotations); err != nil {
			return fmt.Errorf("writing error responses: %w", err)
		}
	}

	// Write composed root
	if err := writer.WriteComposedRoot(doc, config.OpenAPIConfig); err != nil {
		return fmt.Errorf("writing composed root: %w", err)
	}

	// Write bundled dist
	if err := writer.WriteBundledDist(doc); err != nil {
		return fmt.Errorf("writing bundled dist: %w", err)
	}

	return nil
}

func isEntityOrModelSchema(schemaName string, allSchemas map[string]formatdef.Schema) bool {
	// Check if Create/Update/List variants exist
	_, hasCreate := allSchemas[schemaName+"Create"]
	_, hasUpdate := allSchemas[schemaName+"Update"]
	_, hasList := allSchemas[schemaName+"List"]

	return hasCreate || hasUpdate || hasList
}

func groupPathsByResource(paths map[string]formatdef.PathItem, config cfg.OpenAPIConfig) map[string]map[string]formatdef.PathItem {
	grouped := make(map[string]map[string]formatdef.PathItem)

	for path, pathItem := range paths {
		// Extract resource name from path (e.g., "/api/people" -> "people")
		resourceName := extractResourceNameFromPath(path, config)

		if grouped[resourceName] == nil {
			grouped[resourceName] = make(map[string]formatdef.PathItem)
		}

		grouped[resourceName][path] = pathItem
	}

	return grouped
}

func extractResourceNameFromPath(path string, config cfg.OpenAPIConfig) string {
	// Remove base path and extract resource
	// /api/people -> people
	// /api/people/{id} -> people

	if len(path) < len(config.BasePath) {
		return "unknown"
	}

	// Remove base path
	resourcePath := path[len(config.BasePath):]
	if len(resourcePath) > 0 && resourcePath[0] == '/' {
		resourcePath = resourcePath[1:]
	}

	// Extract first segment before {id}
	for i, c := range resourcePath {
		if c == '/' || c == '{' {
			return resourcePath[:i]
		}
	}

	return resourcePath
}

func isDTOSchema(schemaName string) bool {
	return getDTOType(schemaName) != ""
}

func getDTOType(schemaName string) string {
	if len(schemaName) > 6 && schemaName[len(schemaName)-6:] == "Create" {
		return "create"
	}
	if len(schemaName) > 6 && schemaName[len(schemaName)-6:] == "Update" {
		return "update"
	}
	if len(schemaName) > 4 && schemaName[len(schemaName)-4:] == "List" {
		return "list"
	}
	return ""
}

func getBaseName(schemaName string, dtoType string) string {
	switch dtoType {
	case "create":
		return schemaName[:len(schemaName)-6]
	case "update":
		return schemaName[:len(schemaName)-6]
	case "list":
		return schemaName[:len(schemaName)-4]
	}
	return schemaName
}

func getIDStrategy(schema formatdef.Schema) string {
	// Check id field to determine strategy
	if idField, hasID := schema.Properties["id"]; hasID {
		if idField.Format == "uuid" {
			return "uuid"
		}
		if idField.Type == "integer" {
			return "autoincrement"
		}
	}
	if uuidField, hasUUID := schema.Properties["uuid"]; hasUUID {
		if uuidField.Format == "uuid" || uuidField.Type == "string" {
			return "uuid"
		}
	}
	return "string"
}
