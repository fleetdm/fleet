module github.com/fleetdm/fleet/v4

go 1.21.7

require (
	cloud.google.com/go/pubsub v1.36.1
	fyne.io/systray v1.10.1-0.20240111184411-11c585fff98d
	github.com/AbGuthrie/goquery/v2 v2.0.1
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/Masterminds/semver v1.5.0
	github.com/RobotsAndPencils/buford v0.14.0
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/WatchBeam/clock v0.0.0-20170901150240-b08e6b4da7ea
	github.com/XSAM/otelsql v0.10.0
	github.com/andygrunwald/go-jira v1.16.0
	github.com/antchfx/xmlquery v1.3.14
	github.com/aws/aws-sdk-go v1.44.288
	github.com/beevik/etree v1.3.0
	github.com/beevik/ntp v0.3.0
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb
	github.com/briandowns/spinner v1.13.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/clbanning/mxj v1.8.4
	github.com/danieljoos/wincred v1.2.1
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/digitalocean/go-smbios v0.0.0-20180907143718-390a4f403a8e
	github.com/docker/docker v24.0.9+incompatible
	github.com/docker/go-units v0.4.0
	github.com/doug-martin/goqu/v9 v9.18.0
	github.com/e-dard/netbug v0.0.0-20151029172837-e64d308a0b20
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c
	github.com/fatih/color v1.15.0
	github.com/getsentry/sentry-go v0.18.0
	github.com/ghodss/yaml v1.0.0
	github.com/github/smimesign v0.2.0
	github.com/go-git/go-git/v5 v5.11.0
	github.com/go-ini/ini v1.67.0
	github.com/go-kit/kit v0.12.0
	github.com/go-kit/log v0.2.1
	github.com/go-ole/go-ole v1.2.6
	github.com/go-sql-driver/mysql v1.7.1
	github.com/gocarina/gocsv v0.0.0-20220310154401-d4df709ca055
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/gomodule/oauth1 v0.2.0
	github.com/gomodule/redigo v1.8.9
	github.com/google/go-cmp v0.6.0
	github.com/google/go-github/v37 v37.0.0
	github.com/google/uuid v1.6.0
	github.com/goreleaser/goreleaser v1.1.0
	github.com/goreleaser/nfpm/v2 v2.10.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gosuri/uilive v0.0.4
	github.com/groob/finalizer v0.0.0-20170707115354-4c2ed49aabda
	github.com/groob/plist v0.0.0-20220217120414-63fa881b19a5
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/hillu/go-ntdll v0.0.0-20220801201350-0d23f057ef1f
	github.com/igm/sockjs-go/v3 v3.0.2
	github.com/jmoiron/sqlx v1.3.5
	github.com/josephspurrier/goversioninfo v1.4.0
	github.com/kevinburke/go-bindata v3.24.0+incompatible
	github.com/kolide/launcher v1.0.12
	github.com/lib/pq v1.10.9
	github.com/macadmins/osquery-extension v1.0.1
	github.com/mattermost/xml-roundtrip-validator v0.0.0-20201213122252-bcd7e1b9601e
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/micromdm/micromdm v1.9.0
	github.com/mitchellh/go-ps v1.0.0
	github.com/mitchellh/gon v0.2.6-0.20231031204852-2d4f161ccecd
	github.com/mna/redisc v1.3.2
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/ngrok/sqlmw v0.0.0-20211220175533-9d16fdc47b31
	github.com/nukosuke/go-zendesk v0.13.1
	github.com/oklog/run v1.1.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-policy-agent/opa v0.44.0
	github.com/oschwald/geoip2-golang v1.8.0
	github.com/osquery/osquery-go v0.0.0-20231130195733-61ac79279aaa
	github.com/pandatix/nvdapi v0.6.4
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.19.0
	github.com/quasilyte/go-ruleguard/dsl v0.3.22
	github.com/rs/zerolog v1.32.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/saferwall/pe v1.5.2
	github.com/sassoftware/relic/v7 v7.6.2
	github.com/scjalliance/comshim v0.0.0-20230315213746-5e51f40bd3b9
	github.com/sethvargo/go-password v0.2.0
	github.com/shirou/gopsutil/v3 v3.23.3
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.10.0
	github.com/stretchr/testify v1.9.0
	github.com/theupdateframework/go-tuf v0.5.2
	github.com/throttled/throttled/v2 v2.8.0
	github.com/tj/assert v0.0.3
	github.com/ulikunitz/xz v0.5.11
	github.com/urfave/cli/v2 v2.23.5
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	github.com/ziutek/mymysql v1.5.4
	go.elastic.co/apm/module/apmgorilla/v2 v2.3.0
	go.elastic.co/apm/module/apmsql/v2 v2.4.3
	go.elastic.co/apm/v2 v2.4.3
	go.etcd.io/bbolt v1.3.6
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.44.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0
	go.opentelemetry.io/otel/sdk v1.22.0
	golang.org/x/crypto v0.22.0
	golang.org/x/exp v0.0.0-20230105202349-8879d0199aa3
	golang.org/x/image v0.10.0
	golang.org/x/mod v0.12.0
	golang.org/x/net v0.24.0
	golang.org/x/oauth2 v0.16.0
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.19.0
	golang.org/x/text v0.14.0
	golang.org/x/tools v0.13.0
	google.golang.org/api v0.161.0
	google.golang.org/grpc v1.61.0
	gopkg.in/guregu/null.v3 v3.5.0
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v1.0.1
	software.sslmate.com/src/go-pkcs12 v0.4.0
)

require (
	cloud.google.com/go v0.112.0 // indirect
	cloud.google.com/go/compute v1.23.4 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/kms v1.15.6 // indirect
	cloud.google.com/go/storage v1.36.0 // indirect
	code.gitea.io/sdk/gitea v0.15.0 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/AlekSi/pointer v1.2.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/DataDog/zstd v1.5.5 // indirect
	github.com/DisgoOrg/disgohook v1.4.3 // indirect
	github.com/DisgoOrg/log v1.1.0 // indirect
	github.com/DisgoOrg/restclient v1.2.7 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230828082145-3c4c8a2d2371 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/alecthomas/jsonschema v0.0.0-20211022214203-8b29eab41725 // indirect
	github.com/antchfx/xpath v1.2.2 // indirect
	github.com/apache/thrift v0.18.1 // indirect
	github.com/apex/log v1.9.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/atc0005/go-teams-notify/v2 v2.6.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.6 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.27.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/c-bata/go-prompt v0.2.3 // indirect
	github.com/caarlos0/ctrlc v1.0.0 // indirect
	github.com/caarlos0/env/v6 v6.7.0 // indirect
	github.com/caarlos0/go-shellwords v1.0.12 // indirect
	github.com/cavaliercoder/go-cpio v0.0.0-20180626203310-925f9528c45e // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/dghubble/go-twitter v0.0.0-20210609183100-2fdbf421508e // indirect
	github.com/dghubble/oauth1 v0.7.0 // indirect
	github.com/dghubble/sling v1.3.0 // indirect
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-github/v39 v39.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/rpmpack v0.0.0-20210518075352-dc539ef4f2ea // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/goreleaser/chglog v0.1.2 // indirect
	github.com/goreleaser/fileglob v1.2.0 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.0.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/iancoleman/orderedmap v0.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/kolide/kit v0.0.0-20221107170827-fb85e3d59eab // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc6 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.5.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/skeema/knownhosts v1.2.1 // indirect
	github.com/slack-go/slack v0.9.4 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/vartanbeno/go-reddit/v2 v2.0.0 // indirect
	github.com/xanzy/go-gitlab v0.50.3 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yashtewari/glob-intersection v0.1.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.47.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.47.0 // indirect
	go.opentelemetry.io/otel/metric v1.22.0 // indirect
	go.opentelemetry.io/otel/trace v1.22.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	gocloud.dev v0.24.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240205150955-31a09d347014 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240125205218-1f4bbc51befe // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)
