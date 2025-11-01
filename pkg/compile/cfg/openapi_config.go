package cfg

// OpenAPIConfig holds configuration for OpenAPI generation
type OpenAPIConfig struct {
	// BasePath is the base path for all API routes (default "/api")
	BasePath string `json:"basePath"`

	// Servers is a list of server configurations
	Servers []ServerConfig `json:"servers"`

	// Auth defines the authentication scheme
	Auth AuthConfig `json:"auth"`

	// Naming convention for paths and parameters
	Naming string `json:"naming"` // kebab, camel, snake

	// Collections configuration
	Collections CollectionsConfig `json:"collections"`

	// ResourceSource determines what generates CRUD endpoints
	// Options: "entities", "models", "both"
	// Default: "entities"
	ResourceSource string `json:"resourceSource"`

	// ModelsPathsMode controls how model paths are exposed when ResourceSource is "both"
	// Options: "none", "namespaced", "replace_entities"
	// Default: "none"
	ModelsPathsMode string `json:"modelsPathsMode"`

	// ModelsPathsNamespace is the namespace for model paths (e.g., "/_models")
	// Only used when ModelsPathsMode is "namespaced"
	ModelsPathsNamespace string `json:"modelsPathsNamespace"`

	// Pagination configuration
	Pagination PaginationConfig `json:"pagination"`

	// Relations configuration
	Relations RelationsConfig `json:"relations"`

	// IdParam is the parameter name for ID fields (default "id")
	IdParam string `json:"idParam"`

	// ResponseEnvelope wraps responses in {data, meta} structure
	ResponseEnvelope bool `json:"responseEnvelope"`

	// OutputFormat specifies yaml or json (default "yaml")
	OutputFormat string `json:"outputFormat"`

	// SegmentedOutput enables multi-file output mode
	// When true, generates modular fragments in addition to bundled dist/
	SegmentedOutput bool `json:"segmentedOutput"`

	// IncludeAllSchemas includes all enums/structures even if unreferenced
	// Default: false (only include schemas used in API operations)
	IncludeAllSchemas bool `json:"includeAllSchemas"`

	// EmitAnnotations includes kalo-morphe-* metadata annotations
	// Default: true for segmented mode, false for bundled dist/
	EmitAnnotations bool `json:"emitAnnotations"`
}

// ServerConfig defines a server entry
type ServerConfig struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Scheme string `json:"scheme"` // none, bearer, oauth2

	// OAuth2 specific fields
	OAuth2Flows *OAuth2FlowsConfig `json:"oauth2Flows,omitempty"`
}

// OAuth2FlowsConfig defines OAuth2 flow configurations
type OAuth2FlowsConfig struct {
	AuthorizationCode *OAuth2FlowConfig `json:"authorizationCode,omitempty"`
	ClientCredentials *OAuth2FlowConfig `json:"clientCredentials,omitempty"`
	Implicit          *OAuth2FlowConfig `json:"implicit,omitempty"`
	Password          *OAuth2FlowConfig `json:"password,omitempty"`
}

// OAuth2FlowConfig defines a single OAuth2 flow
type OAuth2FlowConfig struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// CollectionsConfig defines collection naming configuration
type CollectionsConfig struct {
	Pluralize bool `json:"pluralize"`
}

// PaginationConfig defines pagination strategy
type PaginationConfig struct {
	Type            string `json:"type"` // cursor, page
	MaxPageSize     int    `json:"maxPageSize"`
	DefaultPageSize int    `json:"defaultPageSize"`
}

// RelationsConfig defines how relations are handled
type RelationsConfig struct {
	Expand bool `json:"expand"` // Whether to expand relations in responses
}

// DefaultOpenAPIConfig returns a default configuration
func DefaultOpenAPIConfig() OpenAPIConfig {
	return OpenAPIConfig{
		BasePath: "/api",
		Servers: []ServerConfig{
			{URL: "http://localhost:8080", Description: "Development server"},
		},
		Auth: AuthConfig{
			Scheme: "none",
		},
		Naming: "kebab",
		Collections: CollectionsConfig{
			Pluralize: true,
		},
		ResourceSource:       "entities",
		ModelsPathsMode:      "none",
		ModelsPathsNamespace: "/_models",
		Pagination: PaginationConfig{
			Type:            "page",
			MaxPageSize:     100,
			DefaultPageSize: 20,
		},
		Relations: RelationsConfig{
			Expand: false,
		},
		IdParam:           "id",
		ResponseEnvelope:  false,
		OutputFormat:      "yaml",
		SegmentedOutput:   false,
		IncludeAllSchemas: false,
		EmitAnnotations:   true,
	}
}
