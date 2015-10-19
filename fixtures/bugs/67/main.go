
package main

import (
    "fmt"
)

// StoreOrder represents an order in this application.
//
// An order can either be created, processed or completed.
//
// swagger:model info
type Info struct {
    // Type of object
    Object string `json:"object" xml:"object"`
    // State of object
    State string `json:"state"`
}

// swagger:route GET /info info getinfo
// Get information
// Responses:
// default: info
func foo() {
    fmt.Println("foo")
}

func main() {
  fmt.Println("show must GO on!!!")
  foo()
}

