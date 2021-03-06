package spec

import (
	"encoding/json"

	"github.com/vikstrous/go-swagger/jsonpointer"
	"github.com/vikstrous/go-swagger/swag"
)

// pathItemProps the path item specific properties
type pathItemProps struct {
	Get        *Operation  `json:"get,omitempty"`
	Put        *Operation  `json:"put,omitempty"`
	Post       *Operation  `json:"post,omitempty"`
	Delete     *Operation  `json:"delete,omitempty"`
	Options    *Operation  `json:"options,omitempty"`
	Head       *Operation  `json:"head,omitempty"`
	Patch      *Operation  `json:"patch,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

// PathItem describes the operations available on a single path.
// A Path Item may be empty, due to [ACL constraints](http://goo.gl/8us55a#securityFiltering).
// The path itself is still exposed to the documentation viewer but they will
// not know which operations and parameters are available.
//
// For more information: http://goo.gl/8us55a#pathItemObject
type PathItem struct {
	refable
	vendorExtensible
	pathItemProps
}

// JSONLookup look up a value by the json property name
func (p PathItem) JSONLookup(token string) (interface{}, error) {
	if ex, ok := p.Extensions[token]; ok {
		return &ex, nil
	}
	if token == "$ref" {
		return &p.Ref, nil
	}
	r, _, err := jsonpointer.GetForToken(p.pathItemProps, token)
	return r, err
}

// UnmarshalJSON hydrates this items instance with the data from JSON
func (p *PathItem) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.refable); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &p.vendorExtensible); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &p.pathItemProps); err != nil {
		return err
	}
	return nil
}

// MarshalJSON converts this items object to JSON
func (p PathItem) MarshalJSON() ([]byte, error) {
	b3, err := json.Marshal(p.refable)
	if err != nil {
		return nil, err
	}
	b4, err := json.Marshal(p.vendorExtensible)
	if err != nil {
		return nil, err
	}
	b5, err := json.Marshal(p.pathItemProps)
	if err != nil {
		return nil, err
	}
	concated := swag.ConcatJSON(b3, b4, b5)
	return concated, nil
}
