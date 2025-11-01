package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	"github.com/kalo-build/go-util/assertfile"
	rcfg "github.com/kalo-build/morphe-go/pkg/registry/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/internal/testutils"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/formatdef"
)

type CompileTestSuite struct {
	assertfile.FileSuite

	TestDirPath            string
	TestGroundTruthDirPath string

	EnumsDirPath      string
	ModelsDirPath     string
	StructuresDirPath string
	EntitiesDirPath   string
}

func TestCompileTestSuite(t *testing.T) {
	suite.Run(t, new(CompileTestSuite))
}

func (suite *CompileTestSuite) SetupTest() {
	suite.TestDirPath = testutils.GetTestDirPath()
	suite.TestGroundTruthDirPath = filepath.Join(suite.TestDirPath, "ground-truth", "compile-minimal")

	suite.EnumsDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "enums")
	suite.ModelsDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "models")
	suite.StructuresDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "structures")
	suite.EntitiesDirPath = filepath.Join(suite.TestDirPath, "registry", "minimal", "entities")
}

func (suite *CompileTestSuite) TearDownTest() {
	suite.TestDirPath = ""
}

func (suite *CompileTestSuite) TestMorpheToOpenAPI() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	outputPath := filepath.Join(workingDirPath, "openapi.yaml")

	// Use defaults and override what's needed
	openapiConfig := cfg.DefaultOpenAPIConfig()
	openapiConfig.ResourceSource = "models"
	openapiConfig.IncludeAllSchemas = false

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: openapiConfig,
		OutputPath:    outputPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)

	suite.NoError(compileErr)

	// Verify the output file exists and compare to ground truth
	suite.FileExists(outputPath)

	// Use the bundled dist/openapi.yaml as ground truth for monolithic mode
	groundTruthPath := filepath.Join(suite.TestGroundTruthDirPath, "dist", "openapi.yaml")
	suite.FileEquals(outputPath, groundTruthPath)

	// Also parse and do structural validation
	generatedData, err := os.ReadFile(outputPath)
	suite.NoError(err)

	var generated formatdef.OpenAPIDocument
	err = yaml.Unmarshal(generatedData, &generated)
	suite.NoError(err, "Failed to parse generated OpenAPI document")

	// Verify document structure
	suite.Equal("3.1.0", generated.OpenAPI, "OpenAPI version should be 3.1.0")
	suite.NotEmpty(generated.Info.Title, "Should have info title")
	suite.NotEmpty(generated.Servers, "Should have servers")

	// Verify components exist
	suite.NotNil(generated.Components)
	suite.NotNil(generated.Components.Schemas)

	// Check that key schemas exist
	suite.Contains(generated.Components.Schemas, "Person")
	suite.Contains(generated.Components.Schemas, "PersonCreate")
	suite.Contains(generated.Components.Schemas, "PersonUpdate")
	suite.Contains(generated.Components.Schemas, "PersonList")
	suite.Contains(generated.Components.Schemas, "Company")
	suite.Contains(generated.Components.Schemas, "CompanyCreate")
	suite.Contains(generated.Components.Schemas, "CompanyUpdate")
	suite.Contains(generated.Components.Schemas, "CompanyList")
	suite.Contains(generated.Components.Schemas, "Nationality")

	// Address should NOT be included (IncludeAllSchemas: false)
	suite.NotContains(generated.Components.Schemas, "Address")
	// UniversalNumber should NOT be included (unreferenced)
	suite.NotContains(generated.Components.Schemas, "UniversalNumber")

	// Verify paths exist
	suite.Contains(generated.Paths, "/api/people")
	suite.Contains(generated.Paths, "/api/people/{id}")
	suite.Contains(generated.Paths, "/api/companies")
	suite.Contains(generated.Paths, "/api/companies/{id}")

	// Verify CRUD operations on people
	peoplePath := generated.Paths["/api/people"]
	suite.NotNil(peoplePath.Get, "GET /api/people should exist")
	suite.NotNil(peoplePath.Post, "POST /api/people should exist")

	personPath := generated.Paths["/api/people/{id}"]
	suite.NotNil(personPath.Get, "GET /api/people/{id} should exist")
	suite.NotNil(personPath.Patch, "PATCH /api/people/{id} should exist")
	suite.NotNil(personPath.Delete, "DELETE /api/people/{id} should exist")

	// Verify tags
	suite.GreaterOrEqual(len(generated.Tags), 2, "Should have at least 2 tags")
	tagNames := make([]string, 0, len(generated.Tags))
	for _, tag := range generated.Tags {
		tagNames = append(tagNames, tag.Name)
	}
	suite.Contains(tagNames, "Person")
	suite.Contains(tagNames, "Company")

	// Verify Person schema structure
	personSchema := generated.Components.Schemas["Person"]
	suite.Equal("object", personSchema.Type)
	suite.Contains(personSchema.Properties, "id")
	suite.Contains(personSchema.Properties, "firstName")
	suite.Contains(personSchema.Properties, "lastName")
	suite.Contains(personSchema.Properties, "nationality")

	// Verify Nationality enum
	nationalitySchema := generated.Components.Schemas["Nationality"]
	suite.Equal("string", nationalitySchema.Type)
	suite.NotNil(nationalitySchema.Enum)
	suite.Len(nationalitySchema.Enum, 3)
}

func (suite *CompileTestSuite) TestMorpheToOpenAPI_JSON() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-json")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	outputPath := filepath.Join(workingDirPath, "openapi.json")

	// Use defaults and override
	openapiConfig := cfg.DefaultOpenAPIConfig()
	openapiConfig.ResourceSource = "models"
	openapiConfig.OutputFormat = "json"

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: openapiConfig,
		OutputPath:    outputPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)

	suite.NoError(compileErr)
	suite.FileExists(outputPath)

	// Verify it's valid JSON
	data, err := os.ReadFile(outputPath)
	suite.NoError(err)

	var doc formatdef.OpenAPIDocument
	err = yaml.Unmarshal(data, &doc)
	suite.NoError(err)
}

func (suite *CompileTestSuite) TestMorpheToOpenAPI_OnlyReferencedSchemas() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-filtered")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	outputPath := filepath.Join(workingDirPath, "openapi.yaml")

	// Use defaults and override
	openapiConfig := cfg.DefaultOpenAPIConfig()
	openapiConfig.ResourceSource = "models"
	openapiConfig.IncludeAllSchemas = false

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: openapiConfig,
		OutputPath:    outputPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Read generated file
	data, err := os.ReadFile(outputPath)
	suite.NoError(err)

	var doc formatdef.OpenAPIDocument
	err = yaml.Unmarshal(data, &doc)
	suite.NoError(err)

	// Nationality is referenced by Person model - should be included
	suite.Contains(doc.Components.Schemas, "Nationality", "Referenced enum should be included")

	// UniversalNumber is not referenced - should NOT be included
	suite.NotContains(doc.Components.Schemas, "UniversalNumber", "Unreferenced enum should be excluded")

	// Address is not referenced - should NOT be included
	suite.NotContains(doc.Components.Schemas, "Address", "Unreferenced structure should be excluded")
}

func (suite *CompileTestSuite) TestMorpheToOpenAPI_IncludeAllSchemas() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-all")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	outputPath := filepath.Join(workingDirPath, "openapi.yaml")

	// Use defaults and override
	openapiConfig := cfg.DefaultOpenAPIConfig()
	openapiConfig.ResourceSource = "models"
	openapiConfig.IncludeAllSchemas = true

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: openapiConfig,
		OutputPath:    outputPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Read generated file
	data, err := os.ReadFile(outputPath)
	suite.NoError(err)

	var doc formatdef.OpenAPIDocument
	err = yaml.Unmarshal(data, &doc)
	suite.NoError(err)

	// All schemas should be included
	suite.Contains(doc.Components.Schemas, "Nationality", "Should include all enums")
	suite.Contains(doc.Components.Schemas, "UniversalNumber", "Should include unreferenced enum when IncludeAllSchemas=true")
	suite.Contains(doc.Components.Schemas, "Address", "Should include unreferenced structure when IncludeAllSchemas=true")
}

func (suite *CompileTestSuite) TestMorpheToOpenAPI_SegmentedOutput() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-segmented")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	// Use defaults and override what's needed
	openapiConfig := cfg.DefaultOpenAPIConfig()
	openapiConfig.ResourceSource = "models"
	openapiConfig.IncludeAllSchemas = false
	openapiConfig.SegmentedOutput = true
	openapiConfig.EmitAnnotations = true

	config := compile.MorpheCompileConfig{
		MorpheLoadRegistryConfig: rcfg.MorpheLoadRegistryConfig{
			RegistryEnumsDirPath:      suite.EnumsDirPath,
			RegistryStructuresDirPath: suite.StructuresDirPath,
			RegistryModelsDirPath:     suite.ModelsDirPath,
			RegistryEntitiesDirPath:   suite.EntitiesDirPath,
		},
		OpenAPIConfig: openapiConfig,
		OutputPath:    workingDirPath,
	}

	compileErr := compile.MorpheToOpenAPI(config)
	suite.NoError(compileErr)

	// Verify directory structure
	enumsDirPath := filepath.Join(workingDirPath, "generated", "enums")
	gtEnumsDirPath := filepath.Join(suite.TestGroundTruthDirPath, "generated", "enums")
	suite.DirExists(enumsDirPath)

	dtosDirPath := filepath.Join(workingDirPath, "generated", "dtos")
	gtDtosDirPath := filepath.Join(suite.TestGroundTruthDirPath, "generated", "dtos")
	suite.DirExists(dtosDirPath)

	entitiesDirPath := filepath.Join(workingDirPath, "generated", "entities")
	gtEntitiesDirPath := filepath.Join(suite.TestGroundTruthDirPath, "generated", "entities")
	suite.DirExists(entitiesDirPath)

	pathsDirPath := filepath.Join(workingDirPath, "generated", "paths")
	gtPathsDirPath := filepath.Join(suite.TestGroundTruthDirPath, "generated", "paths")
	suite.DirExists(pathsDirPath)

	composedDirPath := filepath.Join(workingDirPath, "composed")
	gtComposedDirPath := filepath.Join(suite.TestGroundTruthDirPath, "composed")
	suite.DirExists(composedDirPath)

	distDirPath := filepath.Join(workingDirPath, "dist")
	gtDistDirPath := filepath.Join(suite.TestGroundTruthDirPath, "dist")
	suite.DirExists(distDirPath)

	// Compare enum files
	nationalityPath := filepath.Join(enumsDirPath, "Nationality.enum.yaml")
	gtNationalityPath := filepath.Join(gtEnumsDirPath, "Nationality.enum.yaml")
	suite.FileExists(nationalityPath)
	suite.FileEquals(nationalityPath, gtNationalityPath)

	// Compare entity files
	personEntityPath := filepath.Join(entitiesDirPath, "Person.entity.yaml")
	gtPersonEntityPath := filepath.Join(gtEntitiesDirPath, "Person.entity.yaml")
	suite.FileExists(personEntityPath)
	suite.FileEquals(personEntityPath, gtPersonEntityPath)

	companyEntityPath := filepath.Join(entitiesDirPath, "Company.entity.yaml")
	gtCompanyEntityPath := filepath.Join(gtEntitiesDirPath, "Company.entity.yaml")
	suite.FileExists(companyEntityPath)
	suite.FileEquals(companyEntityPath, gtCompanyEntityPath)

	contactInfoEntityPath := filepath.Join(entitiesDirPath, "ContactInfo.entity.yaml")
	gtContactInfoEntityPath := filepath.Join(gtEntitiesDirPath, "ContactInfo.entity.yaml")
	suite.FileExists(contactInfoEntityPath)
	suite.FileEquals(contactInfoEntityPath, gtContactInfoEntityPath)

	// Compare DTO files
	personCreatePath := filepath.Join(dtosDirPath, "person.create.yaml")
	gtPersonCreatePath := filepath.Join(gtDtosDirPath, "Person.create.yaml")
	suite.FileExists(personCreatePath)
	suite.FileEquals(personCreatePath, gtPersonCreatePath)

	personUpdatePath := filepath.Join(dtosDirPath, "person.update.yaml")
	gtPersonUpdatePath := filepath.Join(gtDtosDirPath, "Person.update.yaml")
	suite.FileExists(personUpdatePath)
	suite.FileEquals(personUpdatePath, gtPersonUpdatePath)

	personListPath := filepath.Join(dtosDirPath, "person.list.yaml")
	gtPersonListPath := filepath.Join(gtDtosDirPath, "Person.list.yaml")
	suite.FileExists(personListPath)
	suite.FileEquals(personListPath, gtPersonListPath)

	// Compare paths files (uses pluralized names)
	personPathsPath := filepath.Join(pathsDirPath, "people.paths.yaml")
	gtPersonPathsPath := filepath.Join(gtPathsDirPath, "people.paths.yaml")
	suite.FileExists(personPathsPath)
	suite.FileEquals(personPathsPath, gtPersonPathsPath)

	// Compare composed root
	composedRootPath := filepath.Join(composedDirPath, "root.yaml")
	gtComposedRootPath := filepath.Join(gtComposedDirPath, "root.yaml")
	suite.FileExists(composedRootPath)
	suite.FileEquals(composedRootPath, gtComposedRootPath)

	// Compare bundled dist
	distPath := filepath.Join(distDirPath, "openapi.yaml")
	gtDistPath := filepath.Join(gtDistDirPath, "openapi.yaml")
	suite.FileExists(distPath)
	suite.FileEquals(distPath, gtDistPath)

	// Verify UniversalNumber is NOT generated (unreferenced)
	universalNumberPath := filepath.Join(enumsDirPath, "UniversalNumber.enum.yaml")
	_, err := os.Stat(universalNumberPath)
	suite.True(os.IsNotExist(err), "Unreferenced enum should not be generated")

	// Verify Address is NOT generated (unreferenced)
	addressPath := filepath.Join(workingDirPath, "generated", "structures", "Address.structure.yaml")
	_, err = os.Stat(addressPath)
	suite.True(os.IsNotExist(err), "Unreferenced structure should not be generated")
}
