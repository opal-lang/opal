module github.com/aledsdavies/devcmd/runtime

go 1.25.0

require (
	github.com/aledsdavies/devcmd/core v0.0.0
	github.com/google/go-cmp v0.7.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/aledsdavies/devcmd/core => ../core
