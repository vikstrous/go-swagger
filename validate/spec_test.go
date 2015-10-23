package validate

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	intvalidate "github.com/vikstrous/go-swagger/internal/validate"
	"github.com/vikstrous/go-swagger/spec"
	"github.com/vikstrous/go-swagger/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestIssue52(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "52", "swagger.json")
	jstext, _ := ioutil.ReadFile(fp)

	// as json schema
	var sch spec.Schema
	if assert.NoError(t, json.Unmarshal(jstext, &sch)) {
		validator := intvalidate.NewSchemaValidator(spec.MustLoadSwagger20Schema(), nil, "", strfmt.Default)
		res := validator.Validate(&sch)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".paths in body is required")
	}

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".paths in body is required")
	}

}

func TestIssue53(t *testing.T) {
	fp := filepath.Join("..", "fixtures", "bugs", "53", "noswagger.json")
	jstext, _ := ioutil.ReadFile(fp)

	// as json schema
	var sch spec.Schema
	if assert.NoError(t, json.Unmarshal(jstext, &sch)) {
		validator := intvalidate.NewSchemaValidator(spec.MustLoadSwagger20Schema(), nil, "", strfmt.Default)
		res := validator.Validate(&sch)
		assert.False(t, res.IsValid())
		assert.EqualError(t, res.Errors[0], ".swagger in body is required")
	}

	// as swagger spec
	doc, err := spec.JSONSpec(fp)
	if assert.NoError(t, err) {
		validator := intvalidate.NewSpecValidator(doc.Schema(), strfmt.Default)
		res, _ := validator.Validate(doc)
		if assert.False(t, res.IsValid()) {
			assert.EqualError(t, res.Errors[0], ".swagger in body is required")
		}
	}
}
