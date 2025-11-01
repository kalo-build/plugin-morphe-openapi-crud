package compile_test

import (
	"testing"

	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/stretchr/testify/suite"
)

type CompileEnumsTestSuite struct {
	suite.Suite
}

func TestCompileEnumsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileEnumsTestSuite))
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_String() {
	enum0 := yaml.Enum{
		Name: "Color",
		Type: yaml.EnumTypeString,
		Entries: map[string]any{
			"Red":   "rgb(255,0,0)",
			"Green": "rgb(0,255,0)",
			"Blue":  "rgb(0,0,255)",
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.Nil(err)
	suite.NotNil(schema)

	suite.Equal("string", schema.Type)
	suite.Len(schema.Enum, 3)
	suite.Contains(schema.Enum, "rgb(255,0,0)")
	suite.Contains(schema.Enum, "rgb(0,255,0)")
	suite.Contains(schema.Enum, "rgb(0,0,255)")
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_Integer() {
	enum0 := yaml.Enum{
		Name: "Analytics",
		Type: yaml.EnumTypeInteger,
		Entries: map[string]any{
			"AnswerToLife":  42,
			"FineStructure": 317,
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.Nil(err)
	suite.NotNil(schema)

	suite.Equal("integer", schema.Type)
	suite.Len(schema.Enum, 2)
	suite.Contains(schema.Enum, 42)
	suite.Contains(schema.Enum, 317)
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_Float() {
	enum0 := yaml.Enum{
		Name: "Analytics",
		Type: yaml.EnumTypeFloat,
		Entries: map[string]any{
			"Pi":    3.141,
			"Euler": 2.718,
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.Nil(err)
	suite.NotNil(schema)

	suite.Equal("number", schema.Type)
	suite.Len(schema.Enum, 2)
	suite.Contains(schema.Enum, 3.141)
	suite.Contains(schema.Enum, 2.718)
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_NoName() {
	enum0 := yaml.Enum{
		Name: "",
		Type: yaml.EnumTypeString,
		Entries: map[string]any{
			"Red": "rgb(255,0,0)",
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.ErrorIs(err, yaml.ErrNoMorpheEnumName)
	suite.Nil(schema)
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_NoType() {
	enum0 := yaml.Enum{
		Name: "Color",
		Type: "",
		Entries: map[string]any{
			"Red": "rgb(255,0,0)",
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.ErrorIs(err, yaml.ErrNoMorpheEnumType)
	suite.Nil(schema)
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_NoEntries() {
	enum0 := yaml.Enum{
		Name:    "Color",
		Type:    yaml.EnumTypeString,
		Entries: map[string]any{},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.ErrorIs(err, yaml.ErrNoMorpheEnumEntries)
	suite.Nil(schema)
}

func (suite *CompileEnumsTestSuite) TestMorpheEnumToSchema_EntryTypeMismatch() {
	enum0 := yaml.Enum{
		Name: "Color",
		Type: yaml.EnumTypeInteger,
		Entries: map[string]any{
			"Red": "rgb(255,0,0)", // String value for integer enum
		},
	}

	schema, err := compile.MorpheEnumToSchema(enum0)

	suite.NotNil(err)
	suite.ErrorContains(err, "does not match the enum type")
	suite.Nil(schema)
}
