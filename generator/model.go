package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/vikstrous/go-swagger/spec"
	"github.com/vikstrous/go-swagger/swag"
)

// GenerateDefinition generates a model file for a schema defintion.
func GenerateDefinition(modelNames []string, includeModel, includeValidator bool, opts GenOpts) error {
	// Load the spec
	specPath, specDoc, err := loadSpec(opts.Spec)
	if err != nil {
		return err
	}

	if len(modelNames) == 0 {
		for k := range specDoc.Spec().Definitions {
			modelNames = append(modelNames, k)
		}
	}

	for _, modelName := range modelNames {
		// lookup schema
		model, ok := specDoc.Spec().Definitions[modelName]
		if !ok {
			return fmt.Errorf("model %q not found in definitions in %s", modelName, specPath)
		}

		// generate files
		generator := definitionGenerator{
			Name:             modelName,
			Model:            model,
			SpecDoc:          specDoc,
			Target:           filepath.Join(opts.Target, opts.ModelPackage),
			IncludeModel:     includeModel,
			IncludeValidator: includeValidator,
			DumpData:         opts.DumpData,
		}

		if err := generator.Generate(); err != nil {
			return err
		}
	}

	return nil
}

type definitionGenerator struct {
	Name             string
	Model            spec.Schema
	SpecDoc          *spec.Document
	Target           string
	IncludeModel     bool
	IncludeValidator bool
	Data             interface{}
	DumpData         bool
}

func (m *definitionGenerator) Generate() error {
	mod, err := makeGenDefinition(m.Name, m.Target, m.Model, m.SpecDoc)
	if err != nil {
		return err
	}
	if m.DumpData {
		bb, _ := json.MarshalIndent(swag.ToDynamicJSON(mod), "", " ")
		fmt.Fprintln(os.Stdout, string(bb))
		return nil
	}

	mod.IncludeValidator = m.IncludeValidator
	m.Data = mod

	if m.IncludeModel {
		if err := m.generateModel(); err != nil {
			return fmt.Errorf("model: %s", err)
		}
	}
	log.Println("generated model", m.Name)

	return nil
}

func (m *definitionGenerator) generateModel() error {
	buf := bytes.NewBuffer(nil)

	if err := modelTemplate.Execute(buf, m.Data); err != nil {
		return err
	}
	log.Println("rendered model template:", m.Name)

	return writeToFile(m.Target, m.Name, buf.Bytes())
}

func makeGenDefinition(name, pkg string, schema spec.Schema, specDoc *spec.Document) (*GenDefinition, error) {
	receiver := "m"
	resolver := &typeResolver{
		ModelsPackage: "",
		ModelName:     name,
		Doc:           specDoc,
	}
	pg := schemaGenContext{
		Path:         "",
		Name:         name,
		Receiver:     receiver,
		IndexVar:     "i",
		ValueExpr:    receiver,
		Schema:       schema,
		Required:     false,
		TypeResolver: resolver,
		Named:        true,
		ExtraSchemas: make(map[string]GenSchema),
	}
	if err := pg.makeGenSchema(); err != nil {
		return nil, err
	}

	var defaultImports []string
	if pg.GenSchema.HasValidations {
		defaultImports = []string{
			"github.com/vikstrous/go-swagger/errors",
			"github.com/vikstrous/go-swagger/strfmt",
			"github.com/vikstrous/go-swagger/httpkit/validate",
		}
	}
	var extras []GenSchema
	for _, v := range pg.ExtraSchemas {
		extras = append(extras, v)
	}

	return &GenDefinition{
		Package:        filepath.Base(pkg),
		GenSchema:      pg.GenSchema,
		DependsOn:      pg.Dependencies,
		DefaultImports: defaultImports,
		ExtraSchemas:   extras,
	}, nil
}

// GenDefinition contains all the properties to generate a
// defintion from a swagger spec
type GenDefinition struct {
	GenSchema
	Package          string
	Imports          map[string]string
	DefaultImports   []string
	ExtraSchemas     []GenSchema
	DependsOn        []string
	IncludeValidator bool
}

// GenSchemaList is a list of schemas for generation.
//
// It can be sorted by name to get a stable struct layout for
// version control and such
type GenSchemaList []GenSchema

func (g GenSchemaList) Len() int           { return len(g) }
func (g GenSchemaList) Swap(i, j int)      { g[i], g[j] = g[j], g[i] }
func (g GenSchemaList) Less(i, j int) bool { return g[i].Name < g[j].Name }

type schemaGenContext struct {
	Path               string
	Name               string
	ParamName          string
	Accessor           string
	Receiver           string
	IndexVar           string
	KeyVar             string
	ValueExpr          string
	Schema             spec.Schema
	Required           bool
	AdditionalProperty bool
	TypeResolver       *typeResolver
	Untyped            bool
	Named              bool
	RefHandled         bool
	Index              int

	GenSchema    GenSchema
	Dependencies []string
	ExtraSchemas map[string]GenSchema
}

func (sg *schemaGenContext) NewSliceBranch(schema *spec.Schema) *schemaGenContext {
	pg := sg.shallowClone()
	indexVar := pg.IndexVar
	if pg.Path == "" {
		pg.Path = "strconv.Itoa(" + indexVar + ")"
	} else {
		pg.Path = pg.Path + "+ \".\" + strconv.Itoa(" + indexVar + ")"
	}
	pg.IndexVar = indexVar + "i"
	pg.ValueExpr = pg.ValueExpr + "[" + indexVar + "]"
	pg.Schema = *schema
	pg.Required = false

	// when this is an anonymous complex object, this needs to become a ref
	return pg
}

func (sg *schemaGenContext) NewAdditionalItems(schema *spec.Schema) *schemaGenContext {
	pg := sg.shallowClone()
	indexVar := pg.IndexVar
	pg.Name = sg.Name + " items"
	itemsLen := 0
	if sg.Schema.Items != nil {
		itemsLen = sg.Schema.Items.Len()
	}
	var mod string
	if itemsLen > 0 {
		mod = "+" + strconv.Itoa(itemsLen)
	}
	if pg.Path == "" {
		pg.Path = "strconv.Itoa(" + indexVar + mod + ")"
	} else {
		pg.Path = pg.Path + "+ \".\" + strconv.Itoa(" + indexVar + mod + ")"
	}
	pg.IndexVar = indexVar
	pg.ValueExpr = sg.ValueExpr + "." + swag.ToGoName(sg.Name) + "Items[" + indexVar + "]"
	pg.Schema = spec.Schema{}
	if schema != nil {
		pg.Schema = *schema
	}
	pg.Required = false
	return pg
}

func (sg *schemaGenContext) NewTupleElement(schema *spec.Schema, index int) *schemaGenContext {
	pg := sg.shallowClone()
	if pg.Path == "" {
		pg.Path = "\"" + strconv.Itoa(index) + "\""
	} else {
		pg.Path = pg.Path + "+ \".\"+\"" + strconv.Itoa(index) + "\""
	}
	pg.ValueExpr = pg.ValueExpr + ".P" + strconv.Itoa(index)
	pg.Required = true
	pg.Schema = *schema
	return pg
}

func (sg *schemaGenContext) NewStructBranch(name string, schema spec.Schema) *schemaGenContext {
	pg := sg.shallowClone()
	if sg.Path == "" {
		pg.Path = fmt.Sprintf("%q", name)
	} else {
		pg.Path = pg.Path + "+\".\"+" + fmt.Sprintf("%q", name)
	}
	pg.Name = name
	pg.ValueExpr = pg.ValueExpr + "." + swag.ToGoName(name)
	pg.Schema = schema
	for _, fn := range sg.Schema.Required {
		if name == fn {
			pg.Required = true
			break
		}
	}
	return pg
}

func (sg *schemaGenContext) shallowClone() *schemaGenContext {
	pg := new(schemaGenContext)
	*pg = *sg
	pg.GenSchema = GenSchema{}
	pg.Dependencies = nil
	pg.Named = false
	pg.Index = 0
	return pg
}

func (sg *schemaGenContext) NewCompositionBranch(schema spec.Schema, index int) *schemaGenContext {
	pg := sg.shallowClone()
	pg.Schema = schema
	pg.Name = "AO" + strconv.Itoa(index)
	if sg.Name != sg.TypeResolver.ModelName {
		pg.Name = sg.Name + pg.Name
	}
	pg.Index = index
	return pg
}

func (sg *schemaGenContext) NewAdditionalProperty(schema spec.Schema) *schemaGenContext {
	pg := sg.shallowClone()
	pg.Schema = schema
	if pg.KeyVar == "" {
		pg.ValueExpr = sg.ValueExpr
	}
	pg.KeyVar += "k"
	pg.ValueExpr += "[" + pg.KeyVar + "]"
	pg.Path = pg.KeyVar
	pg.GenSchema.Suffix = "Value"
	if sg.Path != "" {
		pg.Path = sg.Path + "+\".\"+" + pg.KeyVar
	}
	return pg
}

func (sg *schemaGenContext) schemaValidations() sharedValidations {
	model := sg.Schema

	isRequired := sg.Required
	if sg.Schema.Default != nil || sg.Schema.ReadOnly {
		isRequired = false
	}
	hasNumberValidation := model.Maximum != nil || model.Minimum != nil || model.MultipleOf != nil
	hasStringValidation := model.MaxLength != nil || model.MinLength != nil || model.Pattern != ""
	hasSliceValidations := model.MaxItems != nil || model.MinItems != nil || model.UniqueItems
	hasValidations := isRequired || hasNumberValidation || hasStringValidation || hasSliceValidations

	if len(sg.Schema.Enum) > 0 {
		hasValidations = true
	}

	return sharedValidations{
		Required:            sg.Required,
		Maximum:             model.Maximum,
		ExclusiveMaximum:    model.ExclusiveMaximum,
		Minimum:             model.Minimum,
		ExclusiveMinimum:    model.ExclusiveMinimum,
		MaxLength:           model.MaxLength,
		MinLength:           model.MinLength,
		Pattern:             model.Pattern,
		MaxItems:            model.MaxItems,
		MinItems:            model.MinItems,
		UniqueItems:         model.UniqueItems,
		MultipleOf:          model.MultipleOf,
		Enum:                sg.Schema.Enum,
		HasValidations:      hasValidations,
		HasSliceValidations: hasSliceValidations,
	}
}
func (sg *schemaGenContext) MergeResult(other *schemaGenContext) {
	if other.GenSchema.AdditionalProperties != nil && other.GenSchema.AdditionalProperties.HasValidations {
		sg.GenSchema.HasValidations = true
	}
	if other.GenSchema.HasValidations {
		sg.GenSchema.HasValidations = other.GenSchema.HasValidations
	}
	sg.Dependencies = append(sg.Dependencies, other.Dependencies...)
	for k, v := range other.ExtraSchemas {
		sg.ExtraSchemas[k] = v
	}
}

func (sg *schemaGenContext) buildProperties() error {
	for k, v := range sg.Schema.Properties {
		// check if this requires de-anonymizing, if so lift this as a new struct and extra schema
		tpe, err := sg.TypeResolver.ResolveSchema(&v, true)
		if err != nil {
			return err
		}

		vv := v
		var hasValidations bool
		if tpe.IsComplexObject && tpe.IsAnonymous && len(v.Properties) > 0 {
			pg := sg.makeNewStruct(sg.Name+swag.ToGoName(k), v)
			if sg.Path != "" {
				pg.Path = sg.Path + "+ \".\"+" + fmt.Sprintf("%q", k)
			} else {
				pg.Path = fmt.Sprintf("%q", k)
			}
			if err := pg.makeGenSchema(); err != nil {
				return err
			}
			vv = *spec.RefProperty("#/definitions/" + pg.Name)
			hasValidations = pg.GenSchema.HasValidations
			sg.MergeResult(pg)
			sg.ExtraSchemas[pg.Name] = pg.GenSchema
		}

		emprop := sg.NewStructBranch(k, vv)
		if err := emprop.makeGenSchema(); err != nil {
			return err
		}
		if hasValidations || emprop.GenSchema.HasValidations {
			emprop.GenSchema.HasValidations = true
		}
		sg.MergeResult(emprop)
		sg.GenSchema.Properties = append(sg.GenSchema.Properties, emprop.GenSchema)
	}
	sort.Sort(sg.GenSchema.Properties)
	return nil
}

func (sg *schemaGenContext) buildAllOf() error {
	for i, sch := range sg.Schema.AllOf {
		var comprop *schemaGenContext
		comprop = sg.NewCompositionBranch(sch, i)
		if err := comprop.makeGenSchema(); err != nil {
			return err
		}
		sg.MergeResult(comprop)
		sg.GenSchema.AllOf = append(sg.GenSchema.AllOf, comprop.GenSchema)
	}
	return nil
}

type mapStack struct {
	Type     *spec.Schema
	Next     *mapStack
	Previous *mapStack
	ValueRef *schemaGenContext
	Context  *schemaGenContext
	NewObj   *schemaGenContext
}

func newMapStack(context *schemaGenContext) (first, last *mapStack, err error) {
	ms := &mapStack{
		Type:    &context.Schema,
		Context: context,
	}

	l := ms
	for l.HasMore() {
		tpe, err := l.Context.TypeResolver.ResolveSchema(l.Type.AdditionalProperties.Schema, true)
		if err != nil {
			return nil, nil, err
		}
		if !tpe.IsMap {
			if tpe.IsComplexObject && tpe.IsAnonymous {
				nw := l.Context.makeNewStruct(l.Context.Name+" Anon", *l.Type.AdditionalProperties.Schema)
				sch := spec.RefProperty("#/definitions/" + nw.Name)
				l.NewObj = nw
				l.Type.AdditionalProperties.Schema = sch
				l.ValueRef = l.Context.NewAdditionalProperty(*sch)
			}
			break
		}
		l.Next = &mapStack{
			Previous: l,
			Type:     l.Type.AdditionalProperties.Schema,
			Context:  l.Context.NewAdditionalProperty(*l.Type.AdditionalProperties.Schema),
		}
		l = l.Next
	}

	return ms, l, nil
}

func (mt *mapStack) Build() error {
	if mt.NewObj == nil && mt.ValueRef == nil && mt.Next == nil && mt.Previous == nil {
		cp := mt.Context.NewAdditionalProperty(*mt.Type.AdditionalProperties.Schema)
		if err := cp.makeGenSchema(); err != nil {
			return err
		}
		mt.Context.MergeResult(cp)
		mt.Context.GenSchema.AdditionalProperties = &cp.GenSchema
		return nil
	}
	cur := mt
	for cur != nil {
		if cur.NewObj != nil {
			if err := cur.NewObj.makeGenSchema(); err != nil {
				return err
			}
		}

		if cur.ValueRef != nil {
			if err := cur.ValueRef.makeGenSchema(); err != nil {
				return nil
			}
		}

		if cur.NewObj != nil {
			cur.Context.MergeResult(cur.NewObj)
			cur.Context.ExtraSchemas[cur.NewObj.Name] = cur.NewObj.GenSchema
		}

		if cur.ValueRef != nil {
			if err := cur.Context.makeGenSchema(); err != nil {
				return err
			}
			cur.ValueRef.GenSchema.HasValidations = cur.NewObj.GenSchema.HasValidations
			cur.Context.MergeResult(cur.ValueRef)
			cur.Context.GenSchema.AdditionalProperties = &cur.ValueRef.GenSchema
		}

		if cur.Previous != nil {
			if err := cur.Context.makeGenSchema(); err != nil {
				return err
			}
		}
		if cur.Next != nil {
			cur.Context.MergeResult(cur.Next.Context)
			cur.Context.GenSchema.AdditionalProperties = &cur.Next.Context.GenSchema
		}
		if cur.ValueRef != nil {
			cur.Context.MergeResult(cur.ValueRef)
			cur.Context.GenSchema.AdditionalProperties = &cur.ValueRef.GenSchema
		}
		cur = cur.Previous
	}

	return nil
}

func (mt *mapStack) HasMore() bool {
	return mt.Type.AdditionalProperties != nil && (mt.Type.AdditionalProperties.Allows || mt.Type.AdditionalProperties.Schema != nil)
}

func (mt *mapStack) Dict() map[string]interface{} {
	res := make(map[string]interface{})
	res["context"] = mt.Context.Schema
	if mt.Next != nil {
		res["next"] = mt.Next.Dict()
	}
	if mt.NewObj != nil {
		res["obj"] = mt.NewObj.Schema
	}
	if mt.ValueRef != nil {
		res["value"] = mt.ValueRef.Schema
	}
	return res
}

func (sg *schemaGenContext) buildAdditionalProperties() error {
	if sg.Schema.AdditionalProperties == nil {
		return nil
	}
	addp := *sg.Schema.AdditionalProperties
	wantsAdditional := addp.Allows || addp.Schema != nil
	sg.GenSchema.HasAdditionalProperties = wantsAdditional
	if !wantsAdditional {
		return nil
	}
	// flag swap
	if sg.GenSchema.IsComplexObject {
		sg.GenSchema.IsAdditionalProperties = true
		sg.GenSchema.IsComplexObject = false
		sg.GenSchema.IsMap = false
	}

	if addp.Schema == nil {
		return nil
	}

	if !sg.GenSchema.IsMap && (sg.GenSchema.IsAdditionalProperties && sg.Named) {
		sg.GenSchema.ValueExpression += "." + sg.GenSchema.Name
		comprop := sg.NewAdditionalProperty(*addp.Schema)
		if err := comprop.makeGenSchema(); err != nil {
			return err
		}
		sg.MergeResult(comprop)
		sg.GenSchema.AdditionalProperties = &comprop.GenSchema
		return nil
	}

	if sg.GenSchema.IsMap && wantsAdditional {
		// find out how deep this rabbit hole goes
		// descend, unwind and rewrite
		// This needs to be depth first, so it first goes as deep as it can and then
		// builds the result in reverse order.

		_, ls, err := newMapStack(sg)
		if err != nil {
			return err
		}
		if err := ls.Build(); err != nil {
			return err
		}

		return nil
	}

	if sg.GenSchema.IsAdditionalProperties && !sg.Named {
		// for an anonoymous object, first build the new object
		// and then replace the current one with a $ref to the
		// new object
		newObj := sg.makeNewStruct(sg.GenSchema.Name+" P"+strconv.Itoa(sg.Index), sg.Schema)
		if err := newObj.makeGenSchema(); err != nil {
			return err
		}

		sg.GenSchema = GenSchema{}
		sg.Schema = *spec.RefProperty("#/definitions/" + newObj.Name)
		if err := sg.makeGenSchema(); err != nil {
			return err
		}
		sg.MergeResult(newObj)
		if newObj.GenSchema.HasValidations {
			sg.GenSchema.HasValidations = true
		}
		sg.ExtraSchemas[newObj.Name] = newObj.GenSchema
		return nil
	}
	return nil
}

func (sg *schemaGenContext) makeNewStruct(name string, schema spec.Schema) *schemaGenContext {
	sp := sg.TypeResolver.Doc.Spec()
	name = swag.ToGoName(name)
	if sg.TypeResolver.ModelName != sg.Name {
		name = swag.ToGoName(sg.TypeResolver.ModelName + " " + name)
	}
	sp.Definitions[name] = schema
	pg := schemaGenContext{
		Path:         "",
		Name:         name,
		Receiver:     "m",
		IndexVar:     "i",
		ValueExpr:    "m",
		Schema:       schema,
		Required:     false,
		TypeResolver: sg.TypeResolver,
		Named:        true,
		ExtraSchemas: make(map[string]GenSchema),
	}
	pg.GenSchema.IsVirtual = true

	sg.ExtraSchemas[name] = pg.GenSchema
	return &pg
}

func (sg *schemaGenContext) buildArray() error {
	tpe, err := sg.TypeResolver.ResolveSchema(sg.Schema.Items.Schema, true)
	if err != nil {
		return err
	}
	// check if the element is a complex object, if so generate a new type for it
	if tpe.IsComplexObject && tpe.IsAnonymous {
		pg := sg.makeNewStruct(sg.Name+" items"+strconv.Itoa(sg.Index), *sg.Schema.Items.Schema)
		if err := pg.makeGenSchema(); err != nil {
			return err
		}
		sg.MergeResult(pg)
		sg.ExtraSchemas[pg.Name] = pg.GenSchema
		sg.Schema.Items.Schema = spec.RefProperty("#/definitions/" + pg.Name)
		if err := sg.makeGenSchema(); err != nil {
			return err
		}
		return nil
	}
	elProp := sg.NewSliceBranch(sg.Schema.Items.Schema)
	if err := elProp.makeGenSchema(); err != nil {
		return err
	}
	sg.MergeResult(elProp)
	sg.GenSchema.ItemsEnum = elProp.GenSchema.Enum
	elProp.GenSchema.Suffix = "Items"
	sg.GenSchema.GoType = "[]" + elProp.GenSchema.GoType
	sg.GenSchema.Items = &elProp.GenSchema
	return nil
}

func (sg *schemaGenContext) buildItems() error {
	presentsAsSingle := sg.Schema.Items != nil && sg.Schema.Items.Schema != nil
	if presentsAsSingle && sg.Schema.AdditionalItems != nil { // unsure if htis a valid of invalid schema
		return fmt.Errorf("single schema (%s) can't have additional items", sg.Name)
	}
	if presentsAsSingle {
		return sg.buildArray()
	}
	if sg.Schema.Items == nil {
		return nil
	}
	// This is a tuple, build a new model that represents this
	if sg.Named {
		sg.GenSchema.Name = sg.Name
		sg.GenSchema.GoType = swag.ToGoName(sg.Name)
		if sg.TypeResolver.ModelsPackage != "" {
			sg.GenSchema.GoType = sg.TypeResolver.ModelsPackage + "." + sg.GenSchema.GoType
		}
		for i, s := range sg.Schema.Items.Schemas {
			elProp := sg.NewTupleElement(&s, i)
			if err := elProp.makeGenSchema(); err != nil {
				return err
			}
			sg.MergeResult(elProp)
			elProp.GenSchema.Name = "p" + strconv.Itoa(i)
			sg.GenSchema.Properties = append(sg.GenSchema.Properties, elProp.GenSchema)
		}
		return nil
	}

	// for an anonoymous object, first build the new object
	// and then replace the current one with a $ref to the
	// new tuple object
	var sch spec.Schema
	sch.Typed("object", "")
	sch.Properties = make(map[string]spec.Schema)
	for i, v := range sg.Schema.Items.Schemas {
		sch.Required = append(sch.Required, "P"+strconv.Itoa(i))
		sch.Properties["P"+strconv.Itoa(i)] = v
	}
	sch.AdditionalItems = sg.Schema.AdditionalItems
	tup := sg.makeNewStruct(sg.GenSchema.Name+"Tuple"+strconv.Itoa(sg.Index), sch)
	if err := tup.makeGenSchema(); err != nil {
		return err
	}
	tup.GenSchema.IsTuple = true
	tup.GenSchema.IsComplexObject = false
	tup.GenSchema.Title = tup.GenSchema.Name + " a representation of an anonymous Tuple type"
	tup.GenSchema.Description = ""
	sg.ExtraSchemas[tup.Name] = tup.GenSchema

	sg.Schema = *spec.RefProperty("#/definitions/" + tup.Name)
	if err := sg.makeGenSchema(); err != nil {
		return err
	}
	sg.MergeResult(tup)
	return nil
}

func (sg *schemaGenContext) buildAdditionalItems() error {
	wantsAdditionalItems :=
		sg.Schema.AdditionalItems != nil &&
			(sg.Schema.AdditionalItems.Allows || sg.Schema.AdditionalItems.Schema != nil)

	sg.GenSchema.HasAdditionalItems = wantsAdditionalItems
	if wantsAdditionalItems {
		// check if the element is a complex object, if so generate a new type for it
		tpe, err := sg.TypeResolver.ResolveSchema(sg.Schema.AdditionalItems.Schema, true)
		if err != nil {
			return err
		}
		if tpe.IsComplexObject && tpe.IsAnonymous {
			pg := sg.makeNewStruct(sg.Name+" Items", *sg.Schema.AdditionalItems.Schema)
			if err := pg.makeGenSchema(); err != nil {
				return err
			}
			sg.Schema.AdditionalItems.Schema = spec.RefProperty("#/definitions/" + pg.Name)
			pg.GenSchema.HasValidations = true
			sg.MergeResult(pg)
			sg.ExtraSchemas[pg.Name] = pg.GenSchema
		}

		it := sg.NewAdditionalItems(sg.Schema.AdditionalItems.Schema)
		if tpe.IsInterface {
			it.Untyped = true
		}

		if err := it.makeGenSchema(); err != nil {
			return err
		}
		sg.MergeResult(it)
		sg.GenSchema.AdditionalItems = &it.GenSchema
	}
	return nil
}

func (sg *schemaGenContext) buildXMLName() error {
	if sg.Schema.XML == nil {
		return nil
	}
	sg.GenSchema.XMLName = sg.Name

	if sg.Schema.XML.Name != "" {
		sg.GenSchema.XMLName = sg.Schema.XML.Name
		if sg.Schema.XML.Attribute {
			sg.GenSchema.XMLName += ",attr"
		}
	}
	return nil
}

func (sg *schemaGenContext) shortCircuitNamedRef() (bool, error) {
	// This if block ensures that a struct gets
	// rendered with the ref as embedded ref.
	if sg.RefHandled || !sg.Named || sg.Schema.Ref.GetURL() == nil {
		return false, nil
	}
	nullableOverride := sg.GenSchema.IsNullable
	tpe := resolvedType{}
	tpe.GoType = sg.Name
	if sg.TypeResolver.ModelsPackage != "" {
		tpe.GoType = sg.TypeResolver.ModelsPackage + "." + sg.TypeResolver.ModelName
	}

	tpe.SwaggerType = "object"
	tpe.IsComplexObject = true
	tpe.IsMap = false
	tpe.IsAnonymous = false

	item := sg.NewCompositionBranch(sg.Schema, 0)
	if err := item.makeGenSchema(); err != nil {
		return true, err
	}
	sg.GenSchema.resolvedType = tpe
	sg.GenSchema.IsNullable = sg.GenSchema.IsNullable || nullableOverride
	sg.MergeResult(item)
	sg.GenSchema.AllOf = append(sg.GenSchema.AllOf, item.GenSchema)
	return true, nil
}

func (sg *schemaGenContext) liftSpecialAllOf() error {
	// if there is only a $ref or a primitive and an x-isnullable schema then this is a nullable pointer
	// so this should not compose several objects, just 1
	// if there is a ref with a discriminator then we look for x-class on the current definition to know
	// the value of the discriminator to instantiate the class
	if len(sg.Schema.AllOf) == 0 {
		return nil
	}
	var seenSchema int
	var seenNullable bool
	var schemaToLift spec.Schema

	for _, sch := range sg.Schema.AllOf {

		tpe, err := sg.TypeResolver.ResolveSchema(&sch, true)
		if err != nil {
			return err
		}
		if sg.TypeResolver.isNullable(&sch) {
			seenNullable = true
		}
		if len(sch.Type) > 0 || len(sch.Properties) > 0 || sch.Ref.GetURL() != nil {
			seenSchema++
			if (!tpe.IsAnonymous && tpe.IsComplexObject) || tpe.IsPrimitive {
				schemaToLift = sch
			}
		}
	}

	if seenSchema == 1 {
		sg.Schema = schemaToLift
		sg.GenSchema.IsNullable = seenNullable
	}
	return nil
}

func (sg *schemaGenContext) makeGenSchema() error {
	ex := ""
	if sg.Schema.Example != nil {
		ex = fmt.Sprintf("%#v", sg.Schema.Example)
	}
	sg.GenSchema.Example = ex
	sg.GenSchema.Path = sg.Path
	sg.GenSchema.IndexVar = sg.IndexVar
	sg.GenSchema.Location = "body"
	sg.GenSchema.ValueExpression = sg.ValueExpr
	sg.GenSchema.KeyVar = sg.KeyVar
	sg.GenSchema.Name = sg.Name
	sg.GenSchema.Title = sg.Schema.Title
	sg.GenSchema.Description = sg.Schema.Description
	sg.GenSchema.ReceiverName = sg.Receiver
	sg.GenSchema.sharedValidations = sg.schemaValidations()
	sg.GenSchema.ReadOnly = sg.Schema.ReadOnly

	var err error
	returns, err := sg.shortCircuitNamedRef()
	if err != nil {
		return err
	}
	if returns {
		return nil
	}
	if err := sg.liftSpecialAllOf(); err != nil {
		return err
	}
	nullableOverride := sg.GenSchema.IsNullable

	if err := sg.buildAllOf(); err != nil {
		return err
	}

	var tpe resolvedType
	if sg.Untyped {
		tpe, err = sg.TypeResolver.ResolveSchema(nil, !sg.Named)
	} else {
		tpe, err = sg.TypeResolver.ResolveSchema(&sg.Schema, !sg.Named)
	}
	if err != nil {
		return err
	}
	tpe.IsNullable = tpe.IsNullable || nullableOverride
	sg.GenSchema.resolvedType = tpe

	if err := sg.buildAdditionalProperties(); err != nil {
		return err
	}

	prev := sg.GenSchema
	if sg.Untyped {
		tpe, err = sg.TypeResolver.ResolveSchema(nil, !sg.Named)
	} else {
		tpe, err = sg.TypeResolver.ResolveSchema(&sg.Schema, !sg.Named)
	}
	if err != nil {
		return err
	}
	tpe.IsNullable = tpe.IsNullable || nullableOverride
	sg.GenSchema.resolvedType = tpe
	sg.GenSchema.IsComplexObject = prev.IsComplexObject
	sg.GenSchema.IsMap = prev.IsMap
	sg.GenSchema.IsAdditionalProperties = prev.IsAdditionalProperties

	if err := sg.buildProperties(); err != nil {
		return nil
	}

	if err := sg.buildXMLName(); err != nil {
		return err
	}

	if err := sg.buildAdditionalItems(); err != nil {
		return err
	}

	if err := sg.buildItems(); err != nil {
		return err
	}

	return nil
}

// NOTE:
// untyped data requires a cast somehow to the inner type
// I wonder if this is still a problem after adding support for tuples
// and anonymous structs. At that point there is very little that would
// end up being cast to interface, and if it does it truly is the best guess

// GenSchema contains all the information needed to generate the code
// for a schema
type GenSchema struct {
	resolvedType
	sharedValidations
	Example                 string
	Name                    string
	Suffix                  string
	Path                    string
	ValueExpression         string
	IndexVar                string
	KeyVar                  string
	Title                   string
	Description             string
	Location                string
	ReceiverName            string
	Items                   *GenSchema
	AllowsAdditionalItems   bool
	HasAdditionalItems      bool
	AdditionalItems         *GenSchema
	Object                  *GenSchema
	XMLName                 string
	Properties              GenSchemaList
	AllOf                   []GenSchema
	HasAdditionalProperties bool
	IsAdditionalProperties  bool
	AdditionalProperties    *GenSchema
	ReadOnly                bool
	IsVirtual               bool
}

type sharedValidations struct {
	Required            bool
	MaxLength           *int64
	MinLength           *int64
	Pattern             string
	MultipleOf          *float64
	Minimum             *float64
	Maximum             *float64
	ExclusiveMinimum    bool
	ExclusiveMaximum    bool
	Enum                []interface{}
	ItemsEnum           []interface{}
	HasValidations      bool
	MinItems            *int64
	MaxItems            *int64
	UniqueItems         bool
	HasSliceValidations bool
	NeedsSize           bool
}
