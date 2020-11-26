module github.com/minio/mc

go 1.14

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.9.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/klauspost/compress v1.10.3
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-ieproxy v0.0.1
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/minio/cli v1.22.0
	github.com/minio/minio v0.0.0-20201122074850-39f3d5493bc9
	github.com/minio/minio-go/v7 v7.0.6-0.20201118225257-f6869a5e2a6a
	github.com/minio/sha256-simd v0.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/profile v1.3.0
	github.com/pkg/xattr v0.4.1
	github.com/posener/complete v1.2.3
	github.com/rjeczalik/notify v0.9.2
	github.com/rs/xid v1.2.1
	github.com/shirou/gopsutil v2.20.10-0.20201015215925-32d4603d01b7+incompatible
	golang.org/x/crypto v0.0.0-20201012173705-84dcc777aaee
	golang.org/x/net v0.0.0-20201010224723-4f7140c49acb
	golang.org/x/sys v0.0.0-20201013132646-2da7054afaeb // indirect
	golang.org/x/text v0.3.3
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/h2non/filetype.v1 v1.0.5
	gopkg.in/yaml.v2 v2.2.8
)

replace go.etcd.io/etcd/v3 => go.etcd.io/etcd/v3 v3.3.0-rc.0.0.20200707003333-58bb8ae09f8e
