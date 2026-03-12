package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

// SegmentedWriter handles writing OpenAPI fragments to a modular directory structure
type SegmentedWriter struct {
	BaseOutputPath string
	OutputFormat   string // yaml or json
}

// NewSegmentedWriter creates a new segmented writer
func NewSegmentedWriter(baseOutputPath string, outputFormat string) *SegmentedWriter {
	return &SegmentedWriter{
		BaseOutputPath: baseOutputPath,
		OutputFormat:   outputFormat,
	}
}

// WriteEntitySchema writes an entity schema fragment
func (sw *SegmentedWriter) WriteEntitySchema(entityName string, schema formatdef.Schema, annotations KaloMorpheAnnotations) error {
	return sw.writeSchemaFragment("entities", entityName+".entity", schema, annotations)
}

// WriteDTOSchema writes a DTO schema fragment (Create/Update/List)
func (sw *SegmentedWriter) WriteDTOSchema(resourceName string, dtoType string, schema formatdef.Schema, annotations KaloMorpheAnnotations) error {
	fileName := fmt.Sprintf("%s.%s", strings.ToLower(formatdef.ToKebabCase(resourceName)), dtoType)
	return sw.writeSchemaFragment("dtos", fileName, schema, annotations)
}

// WriteEnumSchema writes an enum schema fragment
func (sw *SegmentedWriter) WriteEnumSchema(enumName string, schema formatdef.Schema, annotations KaloMorpheAnnotations) error {
	return sw.writeSchemaFragment("enums", enumName+".enum", schema, annotations)
}

// WriteStructureSchema writes a structure schema fragment
func (sw *SegmentedWriter) WriteStructureSchema(structureName string, schema formatdef.Schema, annotations KaloMorpheAnnotations) error {
	return sw.writeSchemaFragment("structures", structureName+".structure", schema, annotations)
}

// WritePathsFragment writes a paths fragment for a resource
func (sw *SegmentedWriter) WritePathsFragment(resourceName string, paths map[string]formatdef.PathItem, annotations KaloMorpheAnnotations) error {
	fragment := map[string]interface{}{
		"paths": paths,
	}

	// Add annotations
	for key, value := range annotations {
		fragment[key] = value
	}

	fileName := strings.ToLower(formatdef.ToKebabCase(resourceName)) + ".paths"
	return sw.writeFragment("paths", fileName, fragment)
}

// WriteParametersFragment writes shared parameters
func (sw *SegmentedWriter) WriteParametersFragment(name string, params map[string]formatdef.Parameter, annotations KaloMorpheAnnotations) error {
	fragment := map[string]interface{}{
		"parameters": params,
	}

	// Add annotations
	for key, value := range annotations {
		fragment[key] = value
	}

	return sw.writeFragment("parameters", name+".parameters", fragment)
}

// WriteResponsesFragment writes shared responses
func (sw *SegmentedWriter) WriteResponsesFragment(name string, responses map[string]formatdef.Response, annotations KaloMorpheAnnotations) error {
	fragment := map[string]interface{}{
		"responses": responses,
	}

	// Add annotations
	for key, value := range annotations {
		fragment[key] = value
	}

	return sw.writeFragment("responses", name+".response", fragment)
}

// WriteComposedRoot writes the composed root.yaml that references all fragments
func (sw *SegmentedWriter) WriteComposedRoot(doc *formatdef.OpenAPIDocument, config cfg.OpenAPIConfig) error {
	// Build composed root with references
	root := BuildComposedRoot(doc, config)

	composedPath := filepath.Join(sw.BaseOutputPath, "composed", "root")
	return sw.writeFragmentDirect(composedPath, root)
}

// WriteBundledDist writes the final bundled openapi.yaml (fully dereferenced)
func (sw *SegmentedWriter) WriteBundledDist(doc *formatdef.OpenAPIDocument) error {
	// Strip all kalo-morphe-* annotations for the final dist
	cleanDoc := stripAnnotations(doc)

	distPath := filepath.Join(sw.BaseOutputPath, "dist", "openapi")
	return sw.writeFragmentDirect(distPath, cleanDoc)
}

// writeSchemaFragment writes a schema with annotations
func (sw *SegmentedWriter) writeSchemaFragment(category string, name string, schema formatdef.Schema, annotations KaloMorpheAnnotations) error {
	fragment := map[string]interface{}{
		"schema": schema,
	}

	// Add annotations
	for key, value := range annotations {
		fragment[key] = value
	}

	return sw.writeFragment(category, name, fragment)
}

// writeFragment writes a fragment to the appropriate directory
func (sw *SegmentedWriter) writeFragment(category string, name string, fragment interface{}) error {
	dirPath := filepath.Join(sw.BaseOutputPath, "generated", category)
	filePath := filepath.Join(dirPath, name)

	return sw.writeFragmentDirect(filePath, fragment)
}

// writeFragmentDirect writes a fragment to a specific path
func (sw *SegmentedWriter) writeFragmentDirect(basePath string, fragment interface{}) error {
	// Add extension
	var ext string
	if sw.OutputFormat == "json" {
		ext = ".json"
	} else {
		ext = ".yaml"
	}

	filePath := basePath + ext

	// Ensure directory exists
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dirPath, err)
	}

	// Marshal to bytes
	var data []byte
	var err error

	if sw.OutputFormat == "json" {
		data, err = json.MarshalIndent(fragment, "", "  ")
	} else {
		data, err = yamlv3.Marshal(fragment)
	}

	if err != nil {
		return fmt.Errorf("marshaling fragment: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}

	return nil
}

// stripAnnotations removes all kalo-morphe-* annotations from a document
func stripAnnotations(doc *formatdef.OpenAPIDocument) *formatdef.OpenAPIDocument {
	// For now, return as-is
	// In full implementation, would recursively remove annotation fields
	return doc
}
