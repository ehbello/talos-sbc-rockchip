module rockpi4c

go 1.26.5

require (
	github.com/siderolabs/talos/pkg/machinery v1.13.8
	golang.org/x/sys v0.45.0
)

require go.yaml.in/yaml/v4 v4.0.0-rc.4 // indirect

replace github.com/siderolabs/talos/pkg/machinery => github.com/ehbello/talos/pkg/machinery v1.13.9-0.20260809131157-1ca31dded8e3
