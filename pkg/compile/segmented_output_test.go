package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	yamlv3 "gopkg.in/yaml.v3"

	rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/internal/testutils"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/stretchr/testify/suite"
)

type SegmentedOutputTestSuite struct {
	suite.Suite
	TestDirPath       string
	EnumsDirPath      string
	ModelsDirPath     string
	StructuresDirPath string
	EntitiesDirPath   string
}

func TestSegmentedOutputTestSuite(t *testing.T) {
	suite.Run(t, new(SegmentedOutputTestSuite))
}

func (suite *SegmentedOutputTestSuite) SetupTest() {
	suite.TestDirPath = testutils.GetTestDirPath()
	suite.EnumsDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "enums")
	suite.ModelsDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "models")
	suite.StructuresDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "structures")
	suite.EntitiesDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "entities")
}

func (suite *SegmentedOutputTestSuite) TestSegmentedOutput_EnumFragments() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-seg-enum")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: cfg.OpenAPIConfig{
			BasePath:          "/api",
			Naming:            "kebab",
			ResourceSource:    "models",
			IncludeAllSchemas: false, // Only referenced
			SegmentedOutput:   true,
			EmitAnnotations:   true,
			OutputFormat:      "yaml",
		},
		OutputPath: workingDirPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Verify Nationality enum fragment exists
	nationalityPath := filepath.Join(workingDirPath, "generated", "enums", "Nationality.enum.yaml")
	suite.FileExists(nationalityPath)

	// Verify UniversalNumber is NOT created (unreferenced)
	universalPath := filepath.Join(workingDirPath, "generated", "enums", "UniversalNumber.enum.yaml")
	_, err := os.Stat(universalPath)
	suite.True(os.IsNotExist(err), "Unreferenced enum should not exist")

	// Read and verify Nationality fragment content
	data, err := os.ReadFile(nationalityPath)
	suite.NoError(err)

	var fragment map[string]interface{}
	err = yamlv3.Unmarshal(data, &fragment)
	suite.NoError(err)

	// Check annotations
	suite.Contains(fragment, "kalo-morphe-origin")
	suite.Equal("enum", fragment["kalo-morphe-origin"])
	suite.Contains(fragment, "kalo-morphe-name")
	suite.Equal("Nationality", fragment["kalo-morphe-name"])

	// Check schema is present
	suite.Contains(fragment, "schema")
}

func (suite *SegmentedOutputTestSuite) TestSegmentedOutput_DTOFragments() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-seg-dto")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: cfg.OpenAPIConfig{
			BasePath:        "/api",
			Naming:          "kebab",
			ResourceSource:  "models",
			SegmentedOutput: true,
			EmitAnnotations: true,
			OutputFormat:    "yaml",
		},
		OutputPath: workingDirPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Verify DTO files exist
	suite.FileExists(filepath.Join(workingDirPath, "generated", "dtos", "person.create.yaml"))
	suite.FileExists(filepath.Join(workingDirPath, "generated", "dtos", "person.update.yaml"))
	suite.FileExists(filepath.Join(workingDirPath, "generated", "dtos", "person.list.yaml"))

	// Read PersonCreate DTO
	createPath := filepath.Join(workingDirPath, "generated", "dtos", "person.create.yaml")
	data, err := os.ReadFile(createPath)
	suite.NoError(err)

	var fragment map[string]interface{}
	err = yamlv3.Unmarshal(data, &fragment)
	suite.NoError(err)

	// Verify annotations
	suite.Equal("dto", fragment["kalo-morphe-origin"])
	suite.Contains(fragment, "kalo-morphe-name")
}

func (suite *SegmentedOutputTestSuite) TestSegmentedOutput_DistBundleExists() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-seg-dist")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: cfg.OpenAPIConfig{
			BasePath:        "/api",
			Naming:          "kebab",
			ResourceSource:  "models",
			SegmentedOutput: true,
			OutputFormat:    "yaml",
		},
		OutputPath: workingDirPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Verify bundled dist exists
	distPath := filepath.Join(workingDirPath, "dist", "openapi.yaml")
	suite.FileExists(distPath)

	// Verify it's valid OpenAPI
	data, err := os.ReadFile(distPath)
	suite.NoError(err)
	suite.NotEmpty(data)

	// Should contain the OpenAPI version
	suite.Contains(string(data), "openapi: 3.1.0")
}
