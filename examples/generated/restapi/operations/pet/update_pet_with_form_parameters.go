package pet

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/vikstrous/go-swagger/errors"
	"github.com/vikstrous/go-swagger/httpkit/middleware"
	"github.com/vikstrous/go-swagger/httpkit/validate"
	"github.com/vikstrous/go-swagger/strfmt"
)

// UpdatePetWithFormParams contains all the bound params for the update pet with form operation
// typically these are obtained from a http.Request
type UpdatePetWithFormParams struct {
	// ID of pet that needs to be updated
	PetID string
	// Updated name of the pet
	Name string
	// Updated status of the pet
	Status string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls
func (o *UpdatePetWithFormParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	if err := o.bindPetID(route.Params.Get("petId"), route.Formats); err != nil {
		res = append(res, err)
	}

	if err := o.bindName(r.FormValue("name"), route.Formats); err != nil {
		res = append(res, err)
	}

	if err := o.bindStatus(r.FormValue("status"), route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *UpdatePetWithFormParams) bindPetID(raw string, formats strfmt.Registry) error {

	o.PetID = raw

	if err := o.validatePetID(formats); err != nil {
		return err
	}

	return nil
}

func (o *UpdatePetWithFormParams) validatePetID(formats strfmt.Registry) error {

	return nil
}

func (o *UpdatePetWithFormParams) bindName(raw string, formats strfmt.Registry) error {
	if err := validate.RequiredString("name", "formData", raw); err != nil {
		return err
	}

	o.Name = raw

	if err := o.validateName(formats); err != nil {
		return err
	}

	return nil
}

func (o *UpdatePetWithFormParams) validateName(formats strfmt.Registry) error {

	return nil
}

func (o *UpdatePetWithFormParams) bindStatus(raw string, formats strfmt.Registry) error {
	if err := validate.RequiredString("status", "formData", raw); err != nil {
		return err
	}

	o.Status = raw

	if err := o.validateStatus(formats); err != nil {
		return err
	}

	return nil
}

func (o *UpdatePetWithFormParams) validateStatus(formats strfmt.Registry) error {

	return nil
}
