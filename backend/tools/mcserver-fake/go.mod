module github.com/kofuk/premises/backend/tools/mcserver-fake

go 1.26.1

require (
	github.com/google/uuid v1.6.0
	github.com/kofuk/premises/backend/common v0.0.0-00010101000000-000000000000
)

require (
	github.com/go-logr/logr v1.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/kofuk/premises/backend/common => ../../common
