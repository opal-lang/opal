module github.com/aledsdavies/devcmd/testing

go 1.24.3

require (
	github.com/aledsdavies/devcmd/core v0.0.0
	github.com/aledsdavies/devcmd/runtime v0.0.0
)

replace github.com/aledsdavies/devcmd/core => ../core

replace github.com/aledsdavies/devcmd/runtime => ../runtime
