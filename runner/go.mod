module github.com/kofuk/premises/runner

go 1.21.3

replace github.com/kofuk/premises/common => ../common/

require (
	github.com/google/uuid v1.4.0
	github.com/gorcon/rcon v1.3.4
	github.com/klauspost/compress v1.17.4
	github.com/kofuk/go-mega v0.0.0-20220314143053-5929f3eeeac4
	github.com/kofuk/premises/common v0.0.0-00010101000000-000000000000
	github.com/mackerelio/go-osstat v0.2.4
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
	github.com/ulikunitz/xz v0.5.11
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
