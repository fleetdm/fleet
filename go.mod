module github.com/fleetdm/fleet/v4

go 1.16

require (
	cloud.google.com/go/pubsub v1.16.0
	github.com/AbGuthrie/goquery/v2 v2.0.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/PuerkitoBio/goquery v1.8.0 // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/WatchBeam/clock v0.0.0-20170901150240-b08e6b4da7ea
	github.com/XSAM/otelsql v0.10.0
	github.com/andygrunwald/go-jira v1.15.1 // indirect
	github.com/antchfx/htmlquery v1.2.4 // indirect
	github.com/antchfx/xmlquery v1.3.9 // indirect
	github.com/aws/aws-sdk-go v1.40.34
	github.com/beevik/etree v1.1.0
	github.com/briandowns/spinner v1.13.0
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/dnaeon/go-vcr/v2 v2.0.1
	github.com/doug-martin/goqu/v9 v9.18.0
	github.com/e-dard/netbug v0.0.0-20151029172837-e64d308a0b20
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c // indirect
	github.com/facebookincubator/nvdtools v0.1.4
	github.com/fatih/color v1.12.0
	github.com/fleetdm/goose v0.0.0-20220214194029-91b5e5eb8e77
	github.com/getlantern/golog v0.0.0-20211223150227-d4d95a44d873 // indirect
	github.com/getlantern/hidden v0.0.0-20220104173330-f221c5a24770 // indirect
	github.com/getlantern/ops v0.0.0-20200403153110-8476b16edcd6 // indirect
	github.com/getlantern/systray v1.2.2-0.20220329111105-6065fda28be8
	github.com/getsentry/sentry-go v0.12.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gocarina/gocsv v0.0.0-20220310154401-d4df709ca055
	github.com/gocolly/colly v1.2.0
	github.com/golang-jwt/jwt/v4 v4.3.0
	github.com/gomodule/redigo v1.8.5
	github.com/google/go-cmp v0.5.7
	github.com/google/go-github/v37 v37.0.0
	github.com/google/uuid v1.3.0
	github.com/goreleaser/goreleaser v1.1.0
	github.com/goreleaser/nfpm/v2 v2.10.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gosuri/uilive v0.0.4
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/igm/sockjs-go/v3 v3.0.0
	github.com/jinzhu/copier v0.3.2
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/kevinburke/go-bindata v3.22.0+incompatible
	github.com/kolide/kit v0.0.0-20191023141830-6312ecc11c23
	github.com/kolide/launcher v0.11.25-0.20220321235155-c3e9480037d2
	github.com/macadmins/osquery-extension v0.0.7
	github.com/mattermost/xml-roundtrip-validator v0.0.0-20201213122252-bcd7e1b9601e
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/mitchellh/go-ps v1.0.0
	github.com/mitchellh/gon v0.2.3
	github.com/mna/redisc v1.3.2
	github.com/ngrok/sqlmw v0.0.0-20211220175533-9d16fdc47b31
	github.com/oklog/run v1.1.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-policy-agent/opa v0.24.0
	github.com/oschwald/geoip2-golang v1.6.1
	github.com/osquery/osquery-go v0.0.0-20220317165851-954ac78f381f
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rotisserie/eris v0.5.1
	github.com/rs/zerolog v1.20.0
	github.com/russellhaering/goxmldsig v1.1.0
	github.com/saintfish/chardet v0.0.0-20120816061221-3af4cd4741ca // indirect
	github.com/shirou/gopsutil/v3 v3.22.2
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/temoto/robotstxt v1.1.2 // indirect
	github.com/theupdateframework/go-tuf v0.0.0-20220121203041-e3557e322879
	github.com/throttled/throttled/v2 v2.8.0
	github.com/tj/assert v0.0.3
	github.com/ulikunitz/xz v0.5.10
	github.com/urfave/cli/v2 v2.3.0
	github.com/valyala/fasthttp v1.34.0
	go.elastic.co/apm/module/apmhttp v1.15.0
	go.elastic.co/apm/module/apmsql v1.15.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.28.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220329152356-43be30ef3008
	google.golang.org/grpc v1.42.0
	gopkg.in/guregu/null.v3 v3.4.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/kolide/kit => github.com/zwass/kit v0.0.0-20210625184505-ec5b5c5cce9c
