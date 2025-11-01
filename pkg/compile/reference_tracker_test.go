package compile_test

import (
	"testing"

	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
	"github.com/kalo-build/plugin-morphe-openapi-crud/pkg/compile"
	"github.com/stretchr/testify/suite"
)

type ReferenceTrackerTestSuite struct {
	suite.Suite
}

func TestReferenceTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(ReferenceTrackerTestSuite))
}

func (suite *ReferenceTrackerTestSuite) TestTrackModelReferences_WithEnum() {
	reg := registry.NewRegistry()

	// Add enum to registry
	nationalityEnum := yaml.Enum{
		Name: "Nationality",
		Type: yaml.EnumTypeString,
		Entries: map[string]any{
			"US": "American",
		},
	}
	reg.SetEnum("Nationality", nationalityEnum)

	// Model that references the enum
	model := yaml.Model{
		Name: "Person",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Nationality": {
				Type: "Nationality", // References enum
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {Fields: []string{"ID"}},
		},
	}

	tracker := compile.NewReferenceTracker()
	tracker.TrackModelReferences(reg, model)

	// Should track the Nationality enum
	suite.True(tracker.Enums["Nationality"], "Should track referenced enum")
	suite.True(tracker.ShouldIncludeEnum("Nationality", false), "Should include referenced enum")
}

func (suite *ReferenceTrackerTestSuite) TestTrackModelReferences_NoEnumReferences() {
	reg := registry.NewRegistry()

	// Model with only primitives
	model := yaml.Model{
		Name: "Person",
		Fields: map[string]yaml.ModelField{
			"ID": {
				Type: yaml.ModelFieldTypeAutoIncrement,
			},
			"Name": {
				Type: yaml.ModelFieldTypeString,
			},
		},
		Identifiers: map[string]yaml.ModelIdentifier{
			"primary": {Fields: []string{"ID"}},
		},
	}

	tracker := compile.NewReferenceTracker()
	tracker.TrackModelReferences(reg, model)

	// Should not track any enums
	suite.Empty(tracker.Enums, "Should not track any enums")
}

func (suite *ReferenceTrackerTestSuite) TestShouldIncludeEnum_WithIncludeAll() {
	tracker := compile.NewReferenceTracker()

	// Even unreferenced enums should be included when includeAll is true
	suite.True(tracker.ShouldIncludeEnum("UnusedEnum", true))
}

func (suite *ReferenceTrackerTestSuite) TestShouldIncludeEnum_OnlyReferenced() {
	tracker := compile.NewReferenceTracker()
	tracker.Enums["UsedEnum"] = true

	// Only referenced enums when includeAll is false
	suite.True(tracker.ShouldIncludeEnum("UsedEnum", false))
	suite.False(tracker.ShouldIncludeEnum("UnusedEnum", false))
}

func (suite *ReferenceTrackerTestSuite) TestTrackStructureReferences_Nested() {
	reg := registry.NewRegistry()

	// Nested structure reference
	addressStruct := yaml.Structure{
		Name: "Address",
		Fields: map[string]yaml.StructureField{
			"Street": {Type: yaml.StructureFieldTypeString},
		},
	}
	reg.SetStructure("Address", addressStruct)

	locationStruct := yaml.Structure{
		Name: "Location",
		Fields: map[string]yaml.StructureField{
			"Name":    {Type: yaml.StructureFieldTypeString},
			"Address": {Type: "Address"}, // References Address structure
		},
	}

	tracker := compile.NewReferenceTracker()
	tracker.TrackStructureReferences(reg, locationStruct)

	// Should track nested structure
	suite.True(tracker.Structures["Address"], "Should track nested structure reference")
}
