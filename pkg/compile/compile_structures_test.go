package compile_test

import (
	"testing"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/stretchr/testify/suite"
)

type CompileStructuresTestSuite struct {
	suite.Suite
}

func TestCompileStructuresTestSuite(t *testing.T) {
	suite.Run(t, new(CompileStructuresTestSuite))
}

func (suite *CompileStructuresTestSuite) TestMorpheStructureToSchema() {
	structure0 := yaml.Structure{
		Name: "Address",
		Fields: map[string]yaml.StructureField{
			"Street": {
				Type: yaml.StructureFieldTypeString,
			},
			"HouseNr": {
				Type: yaml.StructureFieldTypeString,
			},
			"ZipCode": {
				Type: yaml.StructureFieldTypeString,
			},
			"City": {
				Type: yaml.StructureFieldTypeString,
			},
		},
	}

	r := registry.NewRegistry()

	schema, err := compile.MorpheStructureToSchema(r, structure0)

	suite.Nil(err)
	suite.NotNil(schema)

	suite.Equal("object", schema.Type)
	suite.Equal("Address structure", schema.Description)
	suite.Len(schema.Properties, 4)
	suite.Len(schema.Required, 4)

	// Check field types
	suite.Equal("string", schema.Properties["City"].Type)
	suite.Equal("string", schema.Properties["HouseNr"].Type)
	suite.Equal("string", schema.Properties["Street"].Type)
	suite.Equal("string", schema.Properties["ZipCode"].Type)
}

func (suite *CompileStructuresTestSuite) TestMorpheStructureToSchema_NoName() {
	structure0 := yaml.Structure{
		Name: "",
		Fields: map[string]yaml.StructureField{
			"Field1": {
				Type: yaml.StructureFieldTypeString,
			},
		},
	}

	r := registry.NewRegistry()

	schema, err := compile.MorpheStructureToSchema(r, structure0)

	suite.NotNil(err)
	suite.ErrorContains(err, "structure has no name")
	suite.Nil(schema)
}

func (suite *CompileStructuresTestSuite) TestMorpheStructureToSchema_NoFields() {
	structure0 := yaml.Structure{
		Name:   "Address",
		Fields: map[string]yaml.StructureField{},
	}

	r := registry.NewRegistry()

	schema, err := compile.MorpheStructureToSchema(r, structure0)

	suite.NotNil(err)
	suite.ErrorContains(err, "has no fields")
	suite.Nil(schema)
}

func (suite *CompileStructuresTestSuite) TestMorpheStructureToSchema_VariousTypes() {
	structure0 := yaml.Structure{
		Name: "Metadata",
		Fields: map[string]yaml.StructureField{
			"Title": {
				Type: yaml.StructureFieldTypeString,
			},
			"Count": {
				Type: yaml.StructureFieldTypeInteger,
			},
			"Score": {
				Type: yaml.StructureFieldTypeFloat,
			},
			"Active": {
				Type: yaml.StructureFieldTypeBoolean,
			},
			"CreatedAt": {
				Type: yaml.StructureFieldTypeTime,
			},
			"BirthDate": {
				Type: yaml.StructureFieldTypeDate,
			},
		},
	}

	r := registry.NewRegistry()

	schema, err := compile.MorpheStructureToSchema(r, structure0)

	suite.Nil(err)
	suite.NotNil(schema)

	suite.Equal("string", schema.Properties["Title"].Type)
	suite.Equal("integer", schema.Properties["Count"].Type)
	suite.Equal("number", schema.Properties["Score"].Type)
	suite.Equal("boolean", schema.Properties["Active"].Type)
	suite.Equal("string", schema.Properties["CreatedAt"].Type)
	suite.Equal("date-time", schema.Properties["CreatedAt"].Format)
	suite.Equal("string", schema.Properties["BirthDate"].Type)
	suite.Equal("date", schema.Properties["BirthDate"].Format)
}
