package operations

import (
	"github.com/vikstrous/go-swagger/fixtures/goparsing/classification/transitive/mods"
	"github.com/vikstrous/go-swagger/strfmt"
)

// A GenericError is an error that is used when no other error is appropriate
// swagger:response genericError
type GenericError struct {
	// The error message
	// in: body
	Body struct {
		Message string
	}
}

// A ValidationError is an error that is used when the required input fails validation.
// swagger:response validationError
type ValidationError struct {
	// The error message
	// in: body
	Body struct {
		// The validation message
		Message string
		// An optional field name to which this validation applies
		FieldName string
	}
}

// A SimpleOne is a model with a few simple fields
type SimpleOne struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int32  `json:"age"`
}

// A ComplexerOne is composed of a SimpleOne and some extra fields.
// swagger:response complexerOne
type ComplexerOne struct {
	SimpleOne
	mods.NotSelected
	mods.Notable
	CreatedAt strfmt.DateTime `json:"createdAt"`
}

// A SomeResponse is a dummy response object to test parsing.
//
// The properties are the same as the other structs used to test parsing.
//
// swagger:response someResponse
type SomeResponse struct {
	// ID of this some response instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// minimum: > 10
	// maximum: < 1000
	ID int64 `json:"id"`

	// The Score of this model
	//
	// minimum: 3
	// maximum: 45
	// multiple of: 3
	Score int32 `json:"score"`

	// Name of this some response instance
	//
	// min length: 4
	// max length: 50
	// pattern: [A-Za-z0-9-.]*
	Name string `json:"x-hdr-name"`

	// Created holds the time when this entry was created
	Created strfmt.DateTime `json:"created"`

	// a FooSlice has foos which are strings
	//
	// min items: 3
	// max items: 10
	// unique: true
	// items.minLength: 3
	// items.maxLength: 10
	// items.pattern: \w+
	// collection format: pipe
	FooSlice []string `json:"foo_slice"`

	// the items for this order
	//
	// in: body
	Items []struct {
		// ID of this some response instance.
		// ids in this application start at 11 and are smaller than 1000
		//
		// required: true
		// minimum: > 10
		// maximum: < 1000
		ID int32 `json:"id"`

		// The Pet to add to this NoModel items bucket.
		// Pets can appear more than once in the bucket
		//
		// required: true
		Pet *mods.Pet `json:"pet"`

		// The amount of pets to add to this bucket.
		//
		// required: true
		// minimum: 1
		// maximum: 10
		Quantity int16 `json:"quantity"`

		// Notes to add to this item.
		// This can be used to add special instructions.
		//
		// required: false
		Notes string `json:"notes"`
	} `json:"items"`
}

type user struct {
	// ID of this some response instance.
	// ids in this application start at 11 and are smaller than 1000
	//
	// required: true
	UserName string `json:"id"`
}

// Resp a response for testing
//
// swagger:response resp
type Resp struct {
	// in: body
	Body *user `json:"user"`
}
