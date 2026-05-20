// Package server is a compile-harness stub.
package server

import "gorm.io/gorm"

// Providers is the dependency bag the boilerplate passes into each feature's
// Register function. In the real project it carries DB, EventBus, adapters,
// etc.; here we only declare DB so generated module.go compiles.
type Providers struct {
	DB *gorm.DB
}
