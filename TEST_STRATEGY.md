# Test Strategy - Morphe OpenAPI Plugin

## ✅ Global Plugin Standard Implemented

Following the established pattern from `plugin-morphe-ts-types`, all integration tests now use **file-by-file ground truth comparison**.

---

## 📁 Ground Truth Structure

```
testdata/
  registry/minimal/           # Input Morphe schemas
    enums/
      nationality.enum
      universal-number.enum   # Unreferenced, NOT in output
    models/
      person.mod
      company.mod
      contact-info.mod
    structures/
      address.str             # Unreferenced, NOT in output
    entities/
      person.ent
      company.ent

  ground-truth/compile-minimal/  # Expected output (20 files)
    generated/
      enums/
        Nationality.enum.yaml       # ✅ Only referenced enum
      entities/
        Person.entity.yaml
        Company.entity.yaml
        ContactInfo.entity.yaml
      dtos/
        Person.{create,update,list}.yaml
        Company.{create,update,list}.yaml
        Contact-Info.{create,update,list}.yaml
      paths/
        people.paths.yaml            # Pluralized!
        companies.paths.yaml
        contact-infos.paths.yaml
      parameters/
        pagination.parameters.yaml
      responses/
        error.response.yaml
    composed/
      root.yaml                      # With $refs
    dist/
      openapi.yaml                   # Bundled, clean
```

---

## 🧪 Test Pattern

### Integration Test Structure

```go
func (suite *CompileTestSuite) TestMorpheToOpenAPI_SegmentedOutput() {
    // 1. Use defaults + override
    openapiConfig := cfg.DefaultOpenAPIConfig()
    openapiConfig.ResourceSource = "models"
    openapiConfig.IncludeAllSchemas = false
    openapiConfig.SegmentedOutput = true
    
    // 2. Compile to working directory
    compile.MorpheToOpenAPI(config)
    
    // 3. Compare each file to ground truth using assertfile
    suite.FileEquals(
        filepath.Join(workingDir, "generated/enums/Nationality.enum.yaml"),
        filepath.Join(groundTruth, "generated/enums/Nationality.enum.yaml"),
    )
    
    // 4. Verify filtered files DON'T exist
    _, err := os.Stat("UniversalNumber.enum.yaml")
    suite.True(os.IsNotExist(err))
}
```

### Key Principles

1. **Use `DefaultOpenAPIConfig()` + override** - Never create zero-value configs
2. **File-by-file comparison** - Use `assertfile.FileEquals()` 
3. **Test both positive and negative** - Verify what exists AND what doesn't
4. **Clean working dirs** - `defer os.RemoveAll(workingDirPath)`

---

## 🎯 What Fixed The Tests

### Before (Broken)
```go
// Missing defaults - IdParam, Collections.Pluralize not set!
OpenAPIConfig: cfg.OpenAPIConfig{
    BasePath: "/api",
    Naming: "kebab",
    // ... incomplete
}
```

**Problems:**
- `IdParam` was empty → paths had `{}` instead of `{id}`
- `Collections.Pluralize` not set → `/api/person` instead of `/api/people`
- Missing servers → file comparison failed

### After (Fixed)
```go
// Start with defaults, then override
openapiConfig := cfg.DefaultOpenAPIConfig()
openapiConfig.ResourceSource = "models"
openapiConfig.IncludeAllSchemas = false
```

**Benefits:**
- ✅ All defaults applied (`IdParam: "id"`, `Collections.Pluralize: true`, etc.)
- ✅ Paths have `{id}` correctly
- ✅ Pluralization works (`people`, `companies`)
- ✅ Servers included
- ✅ File comparison passes

---

## 📊 Test Coverage

| Test Suite | Tests | Purpose |
|------------|-------|---------|
| `CompileEnumsTestSuite` | 7 | Enum → Schema conversion |
| `CompileModelsTestSuite` | 7 | Model → Schemas + CRUD |
| `CompileStructuresTestSuite` | 4 | Structure → Schema |
| `ReferenceTrackerTestSuite` | 5 | Reference filtering logic |
| `SegmentedOutputTestSuite` | 3 | Fragment generation |
| `CompileTestSuite` | 5 | **Integration with file comparison** |

**Total: 31 test cases, all passing**

---

## 🔍 Ground Truth Files Explained

### Why 20 Files?

**3 Models** (Person, Company, ContactInfo) × **4 schemas each** (entity, create, update, list) = 12 files  
**1 Enum** (Nationality, referenced) = 1 file  
**3 Path files** (people, companies, contact-infos) = 3 files  
**1 Parameters** file = 1 file  
**1 Responses** file = 1 file  
**1 Composed** root = 1 file  
**1 Bundled** dist = 1 file  

**Total: 20 files** ✅

### What's NOT Included (By Design)

- ❌ `UniversalNumber.enum.yaml` - Unreferenced enum (filtered)
- ❌ `Address.structure.yaml` - Unreferenced structure (filtered)
- ❌ Old `.go` template files - Deleted

---

## ✅ Final Status

```bash
✅ 31 TEST CASES PASSING
✅ 20 GROUND TRUTH FILES
✅ FILE-BY-FILE COMPARISON WORKING
✅ NO LINTER ERRORS  
✅ NO DEPRECATED CONFIG
✅ PROPER DEFAULT CONFIG USAGE
✅ FOLLOWS PLUGIN STANDARD
```

**Ready to ship!** 🚀

