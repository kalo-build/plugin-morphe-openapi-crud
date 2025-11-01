package formatdef

import (
	"fmt"
	"strings"

	"github.com/kalo-build/go-util/strcase"
)

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	return strcase.ToPascalCase(s)
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	return strcase.ToCamelCase(s)
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	return strcase.ToSnakeCase(s)
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	return strcase.ToKebabCase(s)
}

// ToPlural converts a string to its plural form (simple implementation)
func ToPlural(s string) string {
	// Handle special cases
	lower := strings.ToLower(s)
	if lower == "person" {
		return "people"
	}
	if lower == "child" {
		return "children"
	}

	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		// Check if the letter before 'y' is a consonant
		beforeY := s[len(s)-2]
		if !isVowel(beforeY) {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

func isVowel(c byte) bool {
	vowels := "aeiouAEIOU"
	return strings.ContainsRune(vowels, rune(c))
}

// ConvertName converts a name according to naming convention
func ConvertName(name, naming string, pluralize bool) string {
	var result string
	switch naming {
	case "camel":
		result = ToCamelCase(name)
	case "snake":
		result = ToSnakeCase(name)
	case "kebab":
		result = ToKebabCase(name)
	default:
		result = ToKebabCase(name)
	}

	// Ensure result is lowercase for kebab and snake case
	if naming == "kebab" || naming == "snake" || naming == "" {
		result = strings.ToLower(result)
	}

	if pluralize {
		// For kebab and snake case, we need to pluralize the last segment
		if naming == "kebab" || naming == "" {
			parts := strings.Split(result, "-")
			parts[len(parts)-1] = ToPlural(parts[len(parts)-1])
			return strings.Join(parts, "-")
		} else if naming == "snake" {
			parts := strings.Split(result, "_")
			parts[len(parts)-1] = ToPlural(parts[len(parts)-1])
			return strings.Join(parts, "_")
		}
		return ToPlural(result)
	}

	return result
}

// BuildPath constructs a path from base and segments
func BuildPath(base string, segments ...string) string {
	parts := []string{strings.Trim(base, "/")}
	for _, seg := range segments {
		seg = strings.Trim(seg, "/")
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	result := "/" + strings.Join(parts, "/")
	// Clean up double slashes
	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}
	return result
}

// CreateErrorResponse creates a standard error response schema
func CreateErrorResponse(description string) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"error": {
							Type: "object",
							Properties: map[string]*Schema{
								"code": {
									Type:        "string",
									Description: "Error code",
								},
								"message": {
									Type:        "string",
									Description: "Error message",
								},
							},
							Required: []string{"code", "message"},
						},
					},
					Required: []string{"error"},
				},
			},
		},
	}
}

// CreatePaginatedResponse wraps a schema in pagination metadata
func CreatePaginatedResponse(itemSchemaRef *Schema, description string) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"data": {
							Type:  "array",
							Items: itemSchemaRef,
						},
						"meta": {
							Type: "object",
							Properties: map[string]*Schema{
								"page": {
									Type: "integer",
								},
								"pageSize": {
									Type: "integer",
								},
								"total": {
									Type: "integer",
								},
								"totalPages": {
									Type: "integer",
								},
							},
							Required: []string{"page", "pageSize", "total", "totalPages"},
						},
					},
					Required: []string{"data", "meta"},
				},
			},
		},
	}
}

// CreateEnvelopedResponse wraps a schema in a data envelope
func CreateEnvelopedResponse(schemaRef *Schema, description string) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"data": schemaRef,
					},
					Required: []string{"data"},
				},
			},
		},
	}
}

// CreateSimpleResponse creates a simple response with a schema
func CreateSimpleResponse(schemaRef *Schema, description string) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {
				Schema: schemaRef,
			},
		},
	}
}

// CreatePaginationParameterRefs creates references to pagination parameters in components
func CreatePaginationParameterRefs(paginationType string) []Parameter {
	if paginationType == "cursor" {
		return []Parameter{
			{Ref: "#/components/parameters/cursor"},
			{Ref: "#/components/parameters/limit"},
		}
	}

	// Page-based pagination
	return []Parameter{
		{Ref: "#/components/parameters/page"},
		{Ref: "#/components/parameters/pageSize"},
	}
}

// CreateIDParameter creates a path parameter for ID
func CreateIDParameter(idParam string, idType string, description string) Parameter {
	schema := &Schema{
		Type: idType,
	}

	// Add format hint for integer IDs
	if idType == "integer" {
		schema.Format = "int64"
	} else if idType == "string" {
		// Could be UUID
		schema.Format = "uuid"
	}

	return Parameter{
		Name:        idParam,
		In:          "path",
		Description: description,
		Required:    true,
		Schema:      schema,
	}
}

// CreateOperationID creates a consistent operation ID
func CreateOperationID(tag, operation string) string {
	return fmt.Sprintf("%s_%s", operation, tag)
}
