package commands

import (
	"errors"
	"fmt"

	swaggererrors "github.com/vikstrous/go-swagger/errors"
	"github.com/vikstrous/go-swagger/spec"
	"github.com/vikstrous/go-swagger/strfmt"
	"github.com/vikstrous/go-swagger/validate"
)

// ValidateSpec is a command that validates a swagger document
// against the swagger json schema
type ValidateSpec struct {
	// SchemaURL string `long:"schema" description:"The schema url to use" default:"http://swagger.io/v2/schema.json"`
}

// Execute validates the spec
func (c *ValidateSpec) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("The validate command requires the swagger document url to be specified")
	}

	swaggerDoc := args[0]
	specDoc, err := spec.Load(swaggerDoc)
	if err != nil {
		return nil
	}

	result := validate.Spec(specDoc, strfmt.Default)
	if result == nil {
		fmt.Printf("The swagger spec at %q is valid against swagger specification %s\n", swaggerDoc, specDoc.Version())
	} else {
		str := fmt.Sprintf("The swagger spec at %q is invalid against swagger specification %s. see errors :\n", swaggerDoc, specDoc.Version())
		for _, desc := range result.(*swaggererrors.CompositeError).Errors {
			str += fmt.Sprintf("- %s\n", desc)
		}
		return errors.New(str)
	}
	return nil
}
