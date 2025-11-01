package compile

import (
	"github.com/kalo-build/morphe-go/pkg/registry"
	"github.com/kalo-build/morphe-go/pkg/yaml"
)

// ReferenceTracker tracks which enums and structures are actually referenced
type ReferenceTracker struct {
	Enums      map[string]bool
	Structures map[string]bool
}

// NewReferenceTracker creates a new reference tracker
func NewReferenceTracker() *ReferenceTracker {
	return &ReferenceTracker{
		Enums:      make(map[string]bool),
		Structures: make(map[string]bool),
	}
}

// TrackModelReferences scans a model and tracks enum/structure references
func (rt *ReferenceTracker) TrackModelReferences(reg *registry.Registry, model yaml.Model) {
	for _, field := range model.Fields {
		// Check if field references an enum
		if !yaml.IsModelFieldTypePrimitive(field.Type) {
			enumName := string(field.Type)
			if _, err := reg.GetEnum(enumName); err == nil {
				rt.Enums[enumName] = true
			}
		}
	}
}

// TrackEntityReferences scans an entity and tracks references
func (rt *ReferenceTracker) TrackEntityReferences(reg *registry.Registry, entity yaml.Entity) {
	// Track through the underlying model
	model, err := reg.GetModel(entity.Name)
	if err == nil {
		rt.TrackModelReferences(reg, model)
	}
}

// TrackStructureReferences scans a structure and tracks nested structure references
func (rt *ReferenceTracker) TrackStructureReferences(reg *registry.Registry, structure yaml.Structure) {
	for _, field := range structure.Fields {
		// Check if field references another structure
		if !yaml.IsStructureFieldTypePrimitive(field.Type) {
			structName := string(field.Type)
			if _, err := reg.GetStructure(structName); err == nil {
				rt.Structures[structName] = true
				// Recursively track nested structure references
				if nestedStruct, err := reg.GetStructure(structName); err == nil {
					rt.TrackStructureReferences(reg, nestedStruct)
				}
			}
		}
	}
}

// ShouldIncludeEnum returns true if an enum should be included in the spec
func (rt *ReferenceTracker) ShouldIncludeEnum(enumName string, includeAll bool) bool {
	if includeAll {
		return true
	}
	return rt.Enums[enumName]
}

// ShouldIncludeStructure returns true if a structure should be included in the spec
func (rt *ReferenceTracker) ShouldIncludeStructure(structureName string, includeAll bool) bool {
	if includeAll {
		return true
	}
	return rt.Structures[structureName]
}
