package compile_test

import (
	"testing"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile/cfg"
	"github.com/stretchr/testify/suite"
)

type CompileModelsTestSuite struct {
	suite.Suite
}

func TestCompileModelsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileModelsTestSuite))
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas() {
	model0 := yaml.Model{
		Name: "User",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
			"Email": {
				Type: yaml.ModelFieldTypeString,
			},
			"Age": {
				Type: yaml.ModelFieldTypeInteger,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.Nil(err)
	suite.NotNil(schemas)
	suite.Equal("User", schemas.Name)

	// Check read schema
	suite.NotNil(schemas.ReadSchema)
	suite.Equal("object", schemas.ReadSchema.Type)
	suite.Len(schemas.ReadSchema.Properties, 4)

	// Check create schema (should not include auto-increment ID)
	suite.NotNil(schemas.CreateSchema)
	suite.Equal("object", schemas.CreateSchema.Type)
	suite.NotContains(schemas.CreateSchema.Properties, "id")
	suite.Contains(schemas.CreateSchema.Properties, "name")
	suite.Contains(schemas.CreateSchema.Properties, "email")
	suite.Contains(schemas.CreateSchema.Properties, "age")

	// Check update schema (should not include auto-increment ID)
	suite.NotNil(schemas.UpdateSchema)
	suite.Equal("object", schemas.UpdateSchema.Type)
	suite.NotContains(schemas.UpdateSchema.Properties, "id")

	// Check list schema
	suite.NotNil(schemas.ListSchema)
	suite.Equal("object", schemas.ListSchema.Type)
	suite.Contains(schemas.ListSchema.Properties, "data")
	suite.Contains(schemas.ListSchema.Properties, "meta")
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_WithEnum() {
	enum0 := yaml.Enum{
		Name: "Nationality",
		Type: yaml.EnumTypeString,
		Entries: map[string]any{
			"US": "American",
			"DE": "German",
		},
	}

	model0 := yaml.Model{
		Name: "Person",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
			"Nationality": {
				Type: "Nationality",
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	r.SetEnum("Nationality", enum0)
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.Nil(err)
	suite.NotNil(schemas)

	// Check that nationality references the enum schema
	suite.Contains(schemas.ReadSchema.Properties, "nationality")
	natField := schemas.ReadSchema.Properties["nationality"]
	suite.Equal("#/components/schemas/Nationality", natField.Ref)
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_NoName() {
	model0 := yaml.Model{
		Name: "",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.NotNil(err)
	suite.ErrorContains(err, "has no name")
	suite.Nil(schemas)
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_NoFields() {
	model0 := yaml.Model{
		Name:   "User",
		Fields: map[string]yaml.ModelField{},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.NotNil(err)
	suite.ErrorContains(err, "has no fields")
	suite.Nil(schemas)
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_NoIdentifiers() {
	model0 := yaml.Model{
		Name: "User",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{},
		Related:     map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.NotNil(err)
	suite.ErrorContains(err, "has no identifiers")
	suite.Nil(schemas)
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_WithForOneRelation() {
	parentModel := yaml.Model{
		Name: "Company",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	childModel := yaml.Model{
		Name: "Employee",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"ID"},
			},
		},
		Related: map[string]yaml.ModelRelation{
			"Company": {
				Type: "ForOne",
			},
		},
	}

	r := registry.NewRegistry()
	r.SetModel("Company", parentModel)
	r.SetModel("Employee", childModel)

	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, childModel, config)

	suite.Nil(err)
	suite.NotNil(schemas)

	// Check read schema has company relationship fields
	suite.Contains(schemas.ReadSchema.Properties, "companyID")
	companyIDField := schemas.ReadSchema.Properties["companyID"]
	suite.True(companyIDField.Nullable)

	// Check create schema has companyID
	suite.Contains(schemas.CreateSchema.Properties, "companyID")
}

func (suite *CompileModelsTestSuite) TestMorpheModelToSchemas_WithImmutableField() {
	model0 := yaml.Model{
		Name: "User",
		Fields: map[string]yaml.ModelField{
			"UUID": {
				Type: yaml.ModelFieldTypeUUID,
				Attributes: []string{
					"immutable",
				},
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {
				Fields: []string{"UUID"},
			},
		},
		Related: map[string]yaml.ModelRelation{},
	}

	r := registry.NewRegistry()
	config := cfg.DefaultOpenAPIConfig()

	schemas, err := compile.MorpheModelToSchemas(r, model0, config)

	suite.Nil(err)
	suite.NotNil(schemas)

	// Check read schema has immutable field marked as readOnly
	suite.Contains(schemas.ReadSchema.Properties, "uuid")
	uuidField := schemas.ReadSchema.Properties["uuid"]
	suite.True(uuidField.ReadOnly)

	// Create schema should have uuid (can be provided on create)
	suite.Contains(schemas.CreateSchema.Properties, "uuid")

	// Update schema should not have uuid (immutable)
	suite.NotContains(schemas.UpdateSchema.Properties, "uuid")
}
