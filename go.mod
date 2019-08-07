module github.com/minio/mc

go 1.12

require (
	github.com/cheggaaa/pb v1.0.28
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/gopherjs/gopherjs v0.0.0-20190328170749-bb2674552d8f // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.1
	github.com/mattn/go-isatty v0.0.7
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/minio/cli v1.21.0
	github.com/minio/minio v0.0.0-20190802212500-414a7eca839d
	github.com/minio/minio-go/v6 v6.0.32
	github.com/minio/sha256-simd v0.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/profile v1.3.0
	github.com/pkg/xattr v0.4.1
	github.com/posener/complete v1.2.2-0.20190702141536-6ffe496ea953
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
	github.com/rjeczalik/notify v0.9.2
	github.com/segmentio/go-prompt v1.2.1-0.20161017233205-f0d19b6901ad
	github.com/smartystreets/assertions v0.0.0-20190401211740-f487f9de1cd3 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v1.1.5-pre // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/cheggaaa/pb.v1 v1.0.28 // indirect
	gopkg.in/h2non/filetype.v1 v1.0.5
)

replace github.com/gorilla/rpc v1.2.0+incompatible => github.com/gorilla/rpc v1.2.0
