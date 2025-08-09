module github.com/aledsdavies/devcmd/cli

go 1.24.3

require (
	github.com/aledsdavies/devcmd/core v0.0.0
	github.com/aledsdavies/devcmd/runtime v0.0.0
	github.com/aledsdavies/devcmd/testing v0.0.0-00010101000000-000000000000
	github.com/google/go-cmp v0.7.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
)

replace github.com/aledsdavies/devcmd/core => ../core

replace github.com/aledsdavies/devcmd/runtime => ../runtime

replace github.com/aledsdavies/devcmd/testing => ../testing
