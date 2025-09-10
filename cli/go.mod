module github.com/aledsdavies/devcmd/cli

go 1.24.3

require (
	github.com/aledsdavies/devcmd/core v0.0.0
	github.com/aledsdavies/devcmd/runtime v0.0.0
	github.com/aledsdavies/devcmd/testing v0.0.0
	github.com/google/go-cmp v0.7.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/aledsdavies/devcmd/codegen v0.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/aledsdavies/devcmd/core => ../core

replace github.com/aledsdavies/devcmd/runtime => ../runtime

replace github.com/aledsdavies/devcmd/testing => ../testing

replace github.com/aledsdavies/devcmd/codegen => ../codegen
