package compile

import (
	"fmt"

	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// ComposedRoot represents a root.yaml that references all fragments
type ComposedRoot struct {
	OpenAPI    string                     `json:"openapi" yaml:"openapi"`
	Info       formatdef.Info             `json:"info" yaml:"info"`
	Servers    []formatdef.Server         `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]ComposedPathRef `json:"paths" yaml:"paths"`
	Components ComposedComponents         `json:"components" yaml:"components"`
	Tags       []formatdef.Tag            `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Annotations
	KaloMorpheComposed bool   `json:"kalo-morphe-composed" yaml:"kalo-morphe-composed"`
	KaloMorpheVersion  string `json:"kalo-morphe-version" yaml:"kalo-morphe-version"`
}

// ComposedPathRef represents a reference to a paths fragment
type ComposedPathRef struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// ComposedComponents holds references to component fragments
type ComposedComponents struct {
	Schemas         map[string]ComposedSchemaRef        `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]ComposedResponseRef      `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]ComposedParameterRef     `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	SecuritySchemes map[string]formatdef.SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

// ComposedSchemaRef represents a reference to a schema fragment
type ComposedSchemaRef struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// ComposedResponseRef represents a reference to a response fragment
type ComposedResponseRef struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// ComposedParameterRef represents a reference to a parameter fragment
type ComposedParameterRef struct {
	Ref string `json:"$ref" yaml:"$ref"`
}

// BuildComposedRoot creates a root.yaml with references to all fragments
func BuildComposedRoot(doc *formatdef.OpenAPIDocument, config cfg.OpenAPIConfig) *ComposedRoot {
	root := &ComposedRoot{
		OpenAPI: "3.1.0",
		Info:    doc.Info,
		Servers: doc.Servers,
		Paths:   make(map[string]ComposedPathRef),
		Components: ComposedComponents{
			Schemas:         make(map[string]ComposedSchemaRef),
			Responses:       make(map[string]ComposedResponseRef),
			Parameters:      make(map[string]ComposedParameterRef),
			SecuritySchemes: doc.Components.SecuritySchemes,
		},
		Tags:               doc.Tags,
		KaloMorpheComposed: true,
		KaloMorpheVersion:  "1.0.0",
	}

	// Group paths by resource and create references
	pathsByResource := groupPathsByResource(doc.Paths, config)
	for resourceName := range pathsByResource {
		// Each resource gets its own fragment file
		// Reference format: ./generated/paths/{resource}.paths.yaml#/paths/{path}
		// For now, we'll use a simpler approach of referencing the entire fragment
		kebabName := formatdef.ToKebabCase(resourceName)
		refPath := fmt.Sprintf("./generated/paths/%s.paths.yaml", kebabName)

		// Note: In full implementation, would ref individual paths
		// For now, this is a placeholder showing the structure
		root.Paths["/api/"+resourceName] = ComposedPathRef{
			Ref: refPath,
		}
	}

	// Create schema references
	for schemaName := range doc.Components.Schemas {
		var refPath string

		if isDTOSchema(schemaName) {
			dtoType := getDTOType(schemaName)
			baseName := getBaseName(schemaName, dtoType)
			kebabName := formatdef.ToKebabCase(baseName)
			refPath = fmt.Sprintf("./generated/dtos/%s.%s.yaml#/schema", kebabName, dtoType)
		} else if doc.Components.Schemas[schemaName].Enum != nil {
			refPath = fmt.Sprintf("./generated/enums/%s.enum.yaml#/schema", schemaName)
		} else if isEntityOrModelSchema(schemaName, doc.Components.Schemas) {
			refPath = fmt.Sprintf("./generated/entities/%s.entity.yaml#/schema", schemaName)
		} else {
			refPath = fmt.Sprintf("./generated/structures/%s.structure.yaml#/schema", schemaName)
		}

		root.Components.Schemas[schemaName] = ComposedSchemaRef{Ref: refPath}
	}

	// Create response references
	for respName := range doc.Components.Responses {
		root.Components.Responses[respName] = ComposedResponseRef{
			Ref: "./generated/responses/error.response.yaml#/responses/" + respName,
		}
	}

	// Create parameter references
	for paramName := range doc.Components.Parameters {
		root.Components.Parameters[paramName] = ComposedParameterRef{
			Ref: "./generated/parameters/pagination.parameters.yaml#/parameters/" + paramName,
		}
	}

	return root
}
