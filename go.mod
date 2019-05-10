module github.com/minio/mc

go 1.12

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.0.0-20180919222851-d1e19f5c23e9 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/SAP/go-hdb v0.14.0 // indirect
	github.com/SermoDigital/jose v0.9.1 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20190315122603-6f9e54af456e // indirect
	github.com/araddon/gou v0.0.0-20190110011759-c797efecbb61 // indirect
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf // indirect
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/boombuler/barcode v1.0.0 // indirect
	github.com/briankassouf/jose v0.9.1 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/centrify/cloud-golang-sdk v0.0.0-20190214225812-119110094d0f // indirect
	github.com/cheggaaa/pb v1.0.28
	github.com/chrismalek/oktasdk-go v0.0.0-20181212195951-3430665dfaa0 // indirect
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808 // indirect
	github.com/coreos/go-oidc v2.0.0+incompatible // indirect
	github.com/dancannon/gorethink v4.0.0+incompatible // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190315220205-a8ed825ac853 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/duosecurity/duo_api_golang v0.0.0-20190308151101-6c680f768e74 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fullsailor/pkcs7 v0.0.0-20180613152042-8306686428a5 // indirect
	github.com/gammazero/deque v0.0.0-20190130191400-2afb3858e9c7 // indirect
	github.com/gammazero/workerpool v0.0.0-20181230203049-86a96b5d5d92 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-ldap/ldap v3.0.2+incompatible // indirect
	github.com/go-stomp/stomp v2.0.2+incompatible // indirect
	github.com/go-test/deep v1.0.1 // indirect
	github.com/gocql/gocql v0.0.0-20190301043612-f6df8288f9b4 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/hashicorp/consul v1.4.3 // indirect
	github.com/hashicorp/go-gcp-common v0.0.0-20180425173946-763e39302965 // indirect
	github.com/hashicorp/go-hclog v0.8.0 // indirect
	github.com/hashicorp/go-memdb v0.0.0-20190306140544-eea0b16292ad // indirect
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/nomad v0.8.7 // indirect
	github.com/hashicorp/serf v0.8.2 // indirect
	github.com/hashicorp/vault-plugin-auth-alicloud v0.0.0-20190311155555-98628998247d // indirect
	github.com/hashicorp/vault-plugin-auth-azure v0.0.0-20190201222632-0af1d040b5b3 // indirect
	github.com/hashicorp/vault-plugin-auth-centrify v0.0.0-20180816201131-66b0a34a58bf // indirect
	github.com/hashicorp/vault-plugin-auth-gcp v0.0.0-20190201215414-7d4c2101e7d0 // indirect
	github.com/hashicorp/vault-plugin-auth-jwt v0.0.0-20190314211503-86b44673ce1e // indirect
	github.com/hashicorp/vault-plugin-auth-kubernetes v0.0.0-20190201222209-db96aa4ab438 // indirect
	github.com/hashicorp/vault-plugin-secrets-ad v0.0.0-20190131222416-4796d9980125 // indirect
	github.com/hashicorp/vault-plugin-secrets-alicloud v0.0.0-20190131211812-b0abe36195cb // indirect
	github.com/hashicorp/vault-plugin-secrets-azure v0.0.0-20181207232500-0087bdef705a // indirect
	github.com/hashicorp/vault-plugin-secrets-gcp v0.0.0-20190311200649-621231cb86fe // indirect
	github.com/hashicorp/vault-plugin-secrets-gcpkms v0.0.0-20190116164938-d6b25b0b4a39 // indirect
	github.com/hashicorp/vault-plugin-secrets-kv v0.0.0-20190315192709-dccffee64925 // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/jeffchao/backoff v0.0.0-20140404060208-9d7fd7aa17f2 // indirect
	github.com/jefferai/jsonx v1.0.0 // indirect
	github.com/keybase/go-crypto v0.0.0-20190312101036-b475f2ecc1fe // indirect
	github.com/klauspost/crc32 v0.0.0-20161016154125-cb6bfca970f6 // indirect
	github.com/marstr/guid v1.1.0 // indirect
	github.com/mattbaird/elastigo v0.0.0-20170123220020-2fe47fd29e4b // indirect
	github.com/mattn/go-colorable v0.1.1
	github.com/mattn/go-isatty v0.0.7
	github.com/michaelklishin/rabbit-hole v1.5.0 // indirect
	github.com/minio/cli v1.3.0
	github.com/minio/minio v0.0.0-20190510004154-ac3b59645e92
	github.com/minio/minio-go v0.0.0-20190327203652-5325257a208f
	github.com/minio/sha256-simd v0.0.0-20190328051042-05b4dd3047e5
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/pointerstructure v0.0.0-20170205204203-f2329fcfa9e2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory-am/common v0.4.0 // indirect
	github.com/ory/dockertest v3.3.4+incompatible // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pkg/profile v1.3.0
	github.com/pkg/xattr v0.4.1
	github.com/posener/complete v1.2.1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/pquerna/otp v1.1.0 // indirect
	github.com/rjeczalik/notify v0.9.2
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/samuel/go-zookeeper v0.0.0-20180130194729-c4fab1ac1bec // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/segmentio/go-prompt v1.2.1-0.20161017233205-f0d19b6901ad
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3 // indirect
	golang.org/x/net v0.0.0-20190424112056-4829fb13d2c6
	golang.org/x/text v0.3.2
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/gorethink/gorethink.v4 v4.1.0 // indirect
	gopkg.in/h2non/filetype.v1 v1.0.5
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/ory-am/dockertest.v2 v2.2.3 // indirect
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.0.0-20190313115550-3c12c96769cc // indirect
	k8s.io/apimachinery v0.0.0-20190313115320-c9defaaddf6f // indirect
	k8s.io/klog v0.2.0 // indirect
	layeh.com/radius v0.0.0-20190118135028-0f678f039617 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
