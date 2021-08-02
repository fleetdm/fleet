module github.com/fleetdm/fleet/v4

go 1.16

require (
	cloud.google.com/go/pubsub v1.5.0
	github.com/AbGuthrie/goquery/v2 v2.0.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/WatchBeam/clock v0.0.0-20170901150240-b08e6b4da7ea
	github.com/apache/thrift v0.13.0 // indirect
	github.com/aws/aws-sdk-go v1.36.30
	github.com/beevik/etree v1.1.0
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/briandowns/spinner v0.0.0-20170614154858-48dbb65d7bd5
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dnaeon/go-vcr/v2 v2.0.1
	github.com/e-dard/netbug v0.0.0-20151029172837-e64d308a0b20
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c // indirect
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fatih/color v1.10.0 // indirect
	github.com/fleetdm/goose v0.0.0-20210209032905-c3c01484bacb
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gomodule/redigo v1.8.4
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v37 v37.0.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gosuri/uilive v0.0.4
	github.com/groob/mockimpl v0.0.0-20170306012045-dfa944a2a940 // indirect
	github.com/igm/sockjs-go/v3 v3.0.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/kevinburke/go-bindata v3.22.0+incompatible
	github.com/kolide/kit v0.0.0-20180421083548-36eb8dc43916
	github.com/kolide/launcher v0.0.0-20180427153757-cb412b945cf7
	github.com/kolide/osquery-go v0.0.0-20190904034940-a74aa860032d
	github.com/kr/text v0.2.0 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.0.0-20201213122252-bcd7e1b9601e
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/mna/redisc v1.2.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-policy-agent/opa v0.24.0
	github.com/patrickmn/sortutil v0.0.0-20120526081524-abeda66eb583
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.4.1 // indirect
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v0.0.0-20181021141114-fe5e611709b0
	github.com/spf13/viper v1.8.0
	github.com/stretchr/testify v1.7.0
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
	github.com/throttled/throttled/v2 v2.8.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/tools v0.1.5 // indirect
	google.golang.org/grpc v1.38.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/kolide/kit => github.com/zwass/kit v0.0.0-20210625184505-ec5b5c5cce9c
