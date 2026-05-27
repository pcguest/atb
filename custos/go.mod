module github.com/pcguest/custos

go 1.26.3

// TODO: replace with actual import path before extracting Custos from the ATB workspace.
require github.com/pcguest/atb v0.0.0

require (
	golang.org/x/sys v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/pcguest/atb => ..
