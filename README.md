# Morphe OpenAPI CRUD Plugin

A Morphe compiler plugin that generates OpenAPI 3.1 specifications with automatic CRUD endpoints from Morphe schemas.

## ✨ Features

- **OpenAPI 3.1 compliant** - Generates valid OpenAPI 3.1 documents
- **Convention-based CRUD** - Automatic REST endpoints for models and entities
- **Comprehensive type mapping** - Maps Morphe primitives, enums, and structures to JSON Schema
- **Flexible configuration** - Control naming, pagination, authentication, and more
- **Deterministic output** - Consistent, sorted output for version control
- **JSON or YAML output** - Choose your preferred format

## 🚀 Quick Start

### Basic Usage

```bash
go test ./pkg/compile/... -v
```

### Integration Test

The plugin includes comprehensive tests:

```bash
# Run unit tests
go test ./pkg/compile/compile_enums_test.go -v
go test ./pkg/compile/compile_models_test.go -v
go test ./pkg/compile/compile_structures_test.go -v

# Run integration test
go test ./pkg/compile/compile_test.go -v
```

## 📋 Configuration

Configure via `cfg.OpenAPIConfig`:

```go
config := cfg.OpenAPIConfig{
    BasePath:       "/api",           // API base path
    Naming:         "kebab",          // kebab, camel, or snake
    EntityExposure: "read",           // none, read, or readWrite
    OutputFormat:   "yaml",           // yaml or json
    
    Collections: cfg.CollectionsConfig{
        Pluralize: true,              // Use plural names for collections
    },
    
    Pagination: cfg.PaginationConfig{
        Type:            "page",      // page or cursor
        MaxPageSize:     100,
        DefaultPageSize: 20,
    },
    
    Auth: cfg.AuthConfig{
        Scheme: "bearer",             // none, bearer, or oauth2
    },
}
```

## 🎯 Generated Endpoints

### For Models

Given a `Person` model, generates:

- `GET /api/people` - List with pagination
- `POST /api/people` - Create new person
- `GET /api/people/{id}` - Get single person
- `PATCH /api/people/{id}` - Update person
- `DELETE /api/people/{id}` - Delete person

### For Entities

Based on `entityExposure` setting:
- `none` - No endpoints generated
- `read` - Only GET endpoints
- `readWrite` - Full CRUD

## 📦 Generated Schemas

For each model, creates:

- `{Model}` - Read schema (full object)
- `{Model}Create` - Create request (no auto-generated fields)
- `{Model}Update` - Update request (all fields optional)
- `{Model}List` - Paginated list response

## 🔧 Type Mappings

| Morphe Type        | JSON Schema Type | Format     |
|--------------------|------------------|------------|
| AutoIncrement      | integer          | -          |
| Boolean            | boolean          | -          |
| Date               | string           | date       |
| Float              | number           | double     |
| Integer            | integer          | -          |
| String             | string           | -          |
| Time               | string           | date-time  |
| UUID               | string           | uuid       |
| Protected/Sealed   | string           | writeOnly  |

## 🧪 Testing

The plugin includes:

1. **Unit tests** for each component:
   - `compile_enums_test.go` - Enum schema generation
   - `compile_models_test.go` - Model schema and CRUD paths
   - `compile_structures_test.go` - Structure schema generation

2. **Integration test** (`compile_test.go`):
   - Full end-to-end compilation
   - Validates against ground truth OpenAPI spec
   - Tests both YAML and JSON output

3. **Ground truth** (`testdata/ground-truth/openapi-minimal.yaml`):
   - Reference OpenAPI specification
   - Used for regression testing

## 📝 Example Output

```yaml
openapi: 3.1.0
info:
  title: Generated API
  version: 1.0.0
paths:
  /api/people:
    get:
      summary: List people
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: List of people
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PersonList'
components:
  schemas:
    Person:
      type: object
      properties:
        id:
          type: integer
          readOnly: true
        firstName:
          type: string
        lastName:
          type: string
```

## 🔍 Implementation Details

### Architecture

- `pkg/formatdef/` - OpenAPI type definitions
- `pkg/compile/` - Compilation logic
- `pkg/typemap/` - Morphe to JSON Schema mappings
- `pkg/compile/cfg/` - Configuration structures

### Key Functions

- `MorpheToOpenAPI()` - Main compilation entry point
- `MorpheModelToSchemas()` - Generate model schemas and paths
- `MorpheEnumToSchema()` - Convert enums to JSON Schema
- `MorpheStructureToSchema()` - Convert structures to JSON Schema

## 🤝 Contributing

The plugin follows Morphe plugin conventions:
- No lifecycle hooks (simpler than template)
- Pure function compilation
- Deterministic output
- Comprehensive test coverage

## 📜 License

Same as other Morphe plugins - see LICENSE file.
