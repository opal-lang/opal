module github.com/aledsdavies/devcmd/testing

go 1.24.3

require (
	github.com/aledsdavies/devcmd/core v0.0.0
	github.com/aledsdavies/devcmd/runtime v0.0.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/aledsdavies/devcmd/core => ../core

replace github.com/aledsdavies/devcmd/runtime => ../runtime

replace github.com/aledsdavies/devcmd/codegen => ../codegen
