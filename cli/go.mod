module github.com/aledsdavies/opal/cli

go 1.25.0

require (
	github.com/aledsdavies/opal/core v0.0.0-00010101000000-000000000000
	github.com/aledsdavies/opal/runtime v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/aledsdavies/opal/runtime => ../runtime

replace github.com/aledsdavies/opal/core => ../core
