module github.com/kofuk/premises/runner

go 1.21.3

replace github.com/kofuk/premises/common => ../common/

require (
	github.com/aws/aws-sdk-go-v2 v1.24.1
	github.com/aws/aws-sdk-go-v2/credentials v1.16.13
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.7
	github.com/aws/smithy-go v1.19.0
	github.com/google/uuid v1.5.0
	github.com/gorcon/rcon v1.3.4
	github.com/klauspost/compress v1.17.4
	github.com/kofuk/premises/common v0.0.0-00010101000000-000000000000
	github.com/mackerelio/go-osstat v0.2.4
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
	github.com/ulikunitz/xz v0.5.11
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.9 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
