package pet

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/vikstrous/go-swagger/examples/generated/models"
	"github.com/vikstrous/go-swagger/httpkit/middleware"
)

// UpdatePetWithFormHandlerFunc turns a function with the right signature into a update pet with form handler
type UpdatePetWithFormHandlerFunc func(UpdatePetWithFormParams, *models.User) error

func (fn UpdatePetWithFormHandlerFunc) Handle(params UpdatePetWithFormParams, principal *models.User) error {
	return fn(params, principal)
}

// UpdatePetWithFormHandler interface for that can handle valid update pet with form params
type UpdatePetWithFormHandler interface {
	Handle(UpdatePetWithFormParams, *models.User) error
}

// NewUpdatePetWithForm creates a new http.Handler for the update pet with form operation
func NewUpdatePetWithForm(ctx *middleware.Context, handler UpdatePetWithFormHandler) *UpdatePetWithForm {
	return &UpdatePetWithForm{Context: ctx, Handler: handler}
}

// UpdatePetWithForm
type UpdatePetWithForm struct {
	Context *middleware.Context
	Params  UpdatePetWithFormParams
	Handler UpdatePetWithFormHandler
}

func (o *UpdatePetWithForm) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, _ := o.Context.RouteInfo(r)

	uprinc, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	var principal *models.User
	if uprinc != nil {
		principal = uprinc.(*models.User) // it's ok this is really a models.User
	}

	if err := o.Context.BindValidRequest(r, route, &o.Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	err = o.Handler.Handle(o.Params, principal) // actually handle the request
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	o.Context.Respond(rw, r, route.Produces, route, nil)

}
