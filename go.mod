module github.com/minio/mc

go 1.13

require (
	github.com/cheggaaa/pb v1.0.28
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.4 // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/mattn/go-colorable v0.1.1
	github.com/mattn/go-isatty v0.0.7
	github.com/mattn/go-runewidth v0.0.5 // indirect
	github.com/minio/cli v1.22.0
	github.com/minio/minio v0.0.0-20191205124742-d8e3de0cae46
	github.com/minio/minio-go/v6 v6.0.43
	github.com/minio/sha256-simd v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/profile v1.3.0
	github.com/pkg/xattr v0.4.1
	github.com/posener/complete v1.2.2-0.20190702141536-6ffe496ea953
	github.com/rjeczalik/notify v0.9.2
	github.com/ugorji/go v1.1.7 // indirect
	go.uber.org/zap v1.11.0 // indirect
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.22.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/h2non/filetype.v1 v1.0.5
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/gorilla/rpc v1.2.0+incompatible => github.com/gorilla/rpc v1.2.0
