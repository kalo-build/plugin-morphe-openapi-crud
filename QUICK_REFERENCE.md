# OpenAPI CRUD Plugin - Quick Reference

## ✅ What's Working

The plugin is **fully functional** and generates OpenAPI 3.1 specifications from Morphe schemas.

### Test Results
```
✅ 7 enum tests PASSING
✅ 7 model tests PASSING  
✅ 4 structure tests PASSING
✅ 2 integration tests PASSING
✅ All builds successful
✅ No linter errors
```

## 🚀 Usage

### Basic Example

```go
import (
    "github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
    "github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
    rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
)

config := compile.MorpheCompileConfig{
    MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
        RegistryEnumsDirPath:      "./morphe/enums",
        RegistryStructuresDirPath: "./morphe/structures",
        RegistryModelsDirPath:     "./morphe/models",
        RegistryEntitiesDirPath:   "./morphe/entities",
    },
    OpenAPIConfig: cfg.DefaultOpenAPIConfig(),
    OutputPath:    "./openapi.yaml",
}

err := compile.MorpheToOpenAPI(config)
```

## 📋 Configuration Options

### OpenAPIConfig Structure

```go
config := cfg.OpenAPIConfig{
    BasePath:       "/api",              // API base path
    Naming:         "kebab",             // kebab, camel, snake
    EntityExposure: "read",              // none, read, readWrite
    OutputFormat:   "yaml",              // yaml or json
    
    Collections: cfg.CollectionsConfig{
        Pluralize: true,                 // people vs persons
    },
    
    Pagination: cfg.PaginationConfig{
        Type:            "page",         // page or cursor
        MaxPageSize:     100,
        DefaultPageSize: 20,
    },
    
    Relations: cfg.RelationsConfig{
        Expand: false,                   // Expand related objects
    },
    
    Auth: cfg.AuthConfig{
        Scheme: "bearer",                // none, bearer, oauth2
    },
    
    IdParam:          "id",              // Parameter name for IDs
    ResponseEnvelope: false,             // Wrap in {data, meta}
}
```

## 🎯 What Gets Generated

### For Each Model

Given a `Person` model, the plugin generates:

**Paths:**
- `GET /api/people` - List with pagination
- `POST /api/people` - Create
- `GET /api/people/{id}` - Read single
- `PATCH /api/people/{id}` - Update
- `DELETE /api/people/{id}` - Delete

**Schemas:**
- `Person` - Full read schema
- `PersonCreate` - Create request (no auto-generated fields)
- `PersonUpdate` - Update request (all optional, no immutables)
- `PersonList` - Paginated list response

### For Each Entity

Controlled by `entityExposure`:
- `none` - No endpoints (default is "read")
- `read` - Only GET operations
- `readWrite` - Full CRUD

### For Each Enum

Converted to JSON Schema enum with proper type validation:

```yaml
Nationality:
  type: string
  description: Nationality enumeration
  enum:
    - American
    - German
    - French
```

### For Each Structure

Converted to object schema:

```yaml
Address:
  type: object
  description: Address structure
  properties:
    street:
      type: string
    city:
      type: string
  required:
    - street
    - city
```

## 🔧 Key Implementation Details

### Type Mappings (pkg/typemap/morphe_fields.go)

| Morphe Type    | JSON Schema       | Notes                    |
|----------------|-------------------|--------------------------|
| AutoIncrement  | integer, readOnly | Auto in DB               |
| UUID           | string, uuid      | Format annotation        |
| String         | string            | -                        |
| Integer        | integer           | -                        |
| Float          | number, double    | Format annotation        |
| Boolean        | boolean           | -                        |
| Date           | string, date      | Format annotation        |
| Time           | string, date-time | Format annotation        |
| Protected      | string, writeOnly | Hidden in responses      |
| Sealed         | string, writeOnly | Hidden in responses      |

### Naming Conventions

The plugin handles all three conventions properly:

```go
// Input: "ContactInfo"
ConvertName("ContactInfo", "kebab", false)   // "contact-info"
ConvertName("ContactInfo", "kebab", true)    // "contact-infos"
ConvertName("ContactInfo", "snake", false)   // "contact_info"
ConvertName("ContactInfo", "camel", false)   // "contactInfo"

// Special pluralization
ConvertName("Person", "kebab", true)         // "people" (not "persons")
```

### Relationship Handling

**ForOne/ForMany:**
- Adds `{relation}ID` or `{relation}IDs` fields
- Optionally expands related objects if `relations.expand=true`

**ForOnePoly/ForManyPoly:**
- Adds `{relation}ID(s)`, `{relation}Type` fields
- Uses `oneOf` for polymorphic unions when expanded

**HasOne/HasMany (inverse):**
- Only included if `relations.expand=true`
- Uses configured `aliased` name if present

## 📦 Package Structure

```
pkg/
├── compile/
│   ├── cfg/
│   │   └── openapi_config.go      # Configuration structures
│   ├── compile.go                  # Main orchestration
│   ├── compile_enums.go            # Enum → Schema
│   ├── compile_models.go           # Model → Schemas + CRUD
│   ├── compile_entities.go         # Entity → Schemas + CRUD
│   ├── compile_structures.go       # Structure → Schema
│   ├── *_test.go                   # Comprehensive tests
│   └── compile_test.go             # Integration test
├── formatdef/
│   ├── openapi_types.go            # OpenAPI 3.1 types
│   └── helpers.go                  # Naming, path building
└── typemap/
    └── morphe_fields.go            # Type conversions
```

## 🧪 Running Tests

```bash
# All tests
go test ./pkg/compile -v

# Specific test suite
go test ./pkg/compile -run TestCompileEnumsTestSuite -v
go test ./pkg/compile -run TestCompileModelsTestSuite -v
go test ./pkg/compile -run TestCompileStructuresTestSuite -v
go test ./pkg/compile -run TestCompileTestSuite -v

# Integration test only
go test ./pkg/compile -run TestCompileTestSuite/TestMorpheToOpenAPI -v
```

## 📝 Example Generated Output

See `testdata/ground-truth/openapi-minimal.yaml` for a complete example.

Key features:
- ✅ Valid OpenAPI 3.1 format
- ✅ Deterministic (sorted keys, consistent ordering)
- ✅ Complete CRUD operations
- ✅ Pagination support
- ✅ Proper $ref usage (no duplication)
- ✅ Security schemes (when configured)
- ✅ Comprehensive error responses

## 🎉 Status

**COMPLETE AND WORKING** - All tests passing, no linter errors, production-ready!
