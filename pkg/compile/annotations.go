package compile

// KaloMorpheAnnotations holds metadata annotations for generated OpenAPI components
type KaloMorpheAnnotations map[string]interface{}

// Annotation keys following kalo-morphe-* convention
const (
	AnnotationOrigin        = "kalo-morphe-origin"         // entity | model | enum | structure | dto | response | parameter
	AnnotationName          = "kalo-morphe-name"           // Canonical Morphe name
	AnnotationIDStrategy    = "kalo-morphe-id-strategy"    // uuid | autoincrement | composite
	AnnotationResourceType  = "kalo-morphe-resource-type"  // entity | model
	AnnotationMode          = "kalo-morphe-mode"           // entities | models | both
	AnnotationOperationType = "kalo-morphe-operation-type" // list | create | get | update | delete
	AnnotationResource      = "kalo-morphe-resource"       // entity or model name
	AnnotationShared        = "kalo-morphe-shared"         // true for shared components
	AnnotationStatus        = "kalo-morphe-status"         // preview | stable
	AnnotationComposed      = "kalo-morphe-composed"       // true for composed/root.yaml
	AnnotationVersion       = "kalo-morphe-version"        // plugin version
)

// NewEntityAnnotations creates annotations for an entity schema
func NewEntityAnnotations(entityName string, idStrategy string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationOrigin:       "entity",
		AnnotationName:         entityName,
		AnnotationIDStrategy:   idStrategy,
		AnnotationResourceType: "entity",
	}
}

// NewModelAnnotations creates annotations for a model schema
func NewModelAnnotations(modelName string, idStrategy string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationOrigin:       "model",
		AnnotationName:         modelName,
		AnnotationIDStrategy:   idStrategy,
		AnnotationResourceType: "model",
	}
}

// NewDTOAnnotations creates annotations for a DTO schema (Create/Update/List)
func NewDTOAnnotations(resourceName string, dtoType string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationOrigin: "dto",
		AnnotationName:   resourceName + dtoType,
	}
}

// NewEnumAnnotations creates annotations for an enum schema
func NewEnumAnnotations(enumName string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationOrigin: "enum",
		AnnotationName:   enumName,
	}
}

// NewStructureAnnotations creates annotations for a structure schema
func NewStructureAnnotations(structureName string, referenced bool) KaloMorpheAnnotations {
	annot := KaloMorpheAnnotations{
		AnnotationOrigin: "structure",
		AnnotationName:   structureName,
	}
	if !referenced {
		annot[AnnotationStatus] = "preview"
	}
	return annot
}

// NewPathAnnotations creates annotations for path operations
func NewPathAnnotations(resourceName string, operationType string, resourceType string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationResource:      resourceName,
		AnnotationOperationType: operationType,
		AnnotationResourceType:  resourceType,
	}
}

// NewSharedComponentAnnotations creates annotations for shared components
func NewSharedComponentAnnotations(componentType string) KaloMorpheAnnotations {
	return KaloMorpheAnnotations{
		AnnotationOrigin: componentType,
		AnnotationShared: true,
	}
}
