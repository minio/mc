module github.com/minio/mc

go 1.13

require (
	github.com/cheggaaa/pb v1.0.28
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/klauspost/compress v1.10.3
	github.com/mattn/go-ieproxy v0.0.1
	github.com/mattn/go-isatty v0.0.8
	github.com/minio/cli v1.22.0
	github.com/minio/minio v0.0.0-20200807001021-adcaa6f9de88
	github.com/minio/minio-go/v7 v7.0.5-0.20200807085956-d7db33ea7618
	github.com/minio/sha256-simd v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/profile v1.3.0
	github.com/pkg/xattr v0.4.1
	github.com/posener/complete v1.2.3
	github.com/rjeczalik/notify v0.9.2
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/sys v0.0.0-20200806125547-5acd03effb82 // indirect
	golang.org/x/text v0.3.3
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/h2non/filetype.v1 v1.0.5
	gopkg.in/yaml.v2 v2.2.8
)

replace go.etcd.io/etcd/v3 => go.etcd.io/etcd/v3 v3.3.0-rc.0.0.20200707003333-58bb8ae09f8e
