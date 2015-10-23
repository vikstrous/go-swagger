package pet

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/vikstrous/go-swagger/errors"
	"github.com/vikstrous/go-swagger/httpkit/middleware"
	"github.com/vikstrous/go-swagger/strfmt"
	"github.com/vikstrous/go-swagger/swag"
)

// GetPetByIDParams contains all the bound params for the get pet by i d operation
// typically these are obtained from a http.Request
type GetPetByIDParams struct {
	// ID of pet that needs to be fetched
	PetID int64
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls
func (o *GetPetByIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	if err := o.bindPetID(route.Params.Get("petId"), route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *GetPetByIDParams) bindPetID(raw string, formats strfmt.Registry) error {

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("petId", "path", "int64", raw)
	}
	o.PetID = value

	if err := o.validatePetID(formats); err != nil {
		return err
	}

	return nil
}

func (o *GetPetByIDParams) validatePetID(formats strfmt.Registry) error {

	return nil
}
