module github.com/fleetdm/fleet/v4

go 1.23.1

require (
	cloud.google.com/go/pubsub v1.42.0
	fyne.io/systray v1.10.1-0.20240111184411-11c585fff98d
	github.com/AbGuthrie/goquery/v2 v2.0.1
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/Masterminds/semver v1.5.0
	github.com/RobotsAndPencils/buford v0.14.0
	github.com/VividCortex/mysqlerr v0.0.0-20170204212430-6c6b55f8796f
	github.com/WatchBeam/clock v0.0.0-20170901150240-b08e6b4da7ea
	github.com/XSAM/otelsql v0.35.0
	github.com/andygrunwald/go-jira v1.16.0
	github.com/antchfx/xmlquery v1.3.14
	github.com/apex/log v1.9.0
	github.com/aws/aws-sdk-go v1.55.5
	github.com/beevik/etree v1.3.0
	github.com/beevik/ntp v0.3.0
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb
	github.com/boltdb/bolt v1.3.1
	github.com/briandowns/spinner v1.23.1
	github.com/cavaliergopher/rpm v1.2.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/clbanning/mxj v1.8.4
	github.com/danieljoos/wincred v1.2.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dgraph-io/badger/v2 v2.2007.4
	github.com/digitalocean/go-smbios v0.0.0-20180907143718-390a4f403a8e
	github.com/docker/docker v27.3.1+incompatible
	github.com/docker/go-units v0.5.0
	github.com/doug-martin/goqu/v9 v9.18.0
	github.com/e-dard/netbug v0.0.0-20151029172837-e64d308a0b20
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/facebookincubator/flog v0.0.0-20190930132826-d2511d0ce33c
	github.com/fatih/color v1.16.0
	github.com/getsentry/sentry-go v0.18.0
	github.com/ghodss/yaml v1.0.0
	github.com/github/smimesign v0.2.0
	github.com/go-git/go-git/v5 v5.12.0
	github.com/go-ini/ini v1.67.0
	github.com/go-kit/kit v0.12.0
	github.com/go-kit/log v0.2.1
	github.com/go-ole/go-ole v1.2.6
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gocarina/gocsv v0.0.0-20220310154401-d4df709ca055
	github.com/golang-jwt/jwt/v4 v4.5.1
	github.com/gomodule/oauth1 v0.2.0
	github.com/gomodule/redigo v1.8.9
	github.com/google/go-cmp v0.6.0
	github.com/google/go-github/v37 v37.0.0
	github.com/google/uuid v1.6.0
	github.com/goreleaser/goreleaser/v2 v2.4.4
	github.com/goreleaser/nfpm/v2 v2.41.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
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
	github.com/macadmins/osquery-extension v1.2.1
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
	github.com/open-policy-agent/opa v0.68.0
	github.com/oschwald/geoip2-golang v1.8.0
	github.com/osquery/osquery-go v0.0.0-20231130195733-61ac79279aaa
	github.com/pandatix/nvdapi v0.6.4
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.20.2
	github.com/quasilyte/go-ruleguard/dsl v0.3.22
	github.com/rs/zerolog v1.32.0
	github.com/russellhaering/goxmldsig v1.2.0
	github.com/saferwall/pe v1.5.2
	github.com/sassoftware/relic/v8 v8.0.1
	github.com/scjalliance/comshim v0.0.0-20230315213746-5e51f40bd3b9
	github.com/sethvargo/go-password v0.3.0
	github.com/shirou/gopsutil/v3 v3.24.3
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/smallstep/pkcs7 v0.0.0-20240723090913-5e2c6a136dfa
	github.com/smallstep/scep v0.0.0-20240214080410-892e41795b99
	github.com/spf13/cast v1.6.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	github.com/theupdateframework/go-tuf v0.7.0
	github.com/throttled/throttled/v2 v2.8.0
	github.com/tj/assert v0.0.3
	github.com/ulikunitz/xz v0.5.12
	github.com/urfave/cli/v2 v2.25.7
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	github.com/ziutek/mymysql v1.5.4
	go.elastic.co/apm/module/apmgorilla/v2 v2.3.0
	go.elastic.co/apm/module/apmsql/v2 v2.4.3
	go.elastic.co/apm/v2 v2.4.3
	go.etcd.io/bbolt v1.3.9
	go.mozilla.org/pkcs7 v0.0.0-20210826202110-33d05740a352
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.56.0
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.31.0
	go.opentelemetry.io/otel/sdk v1.31.0
	golang.org/x/crypto v0.28.0
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/image v0.18.0
	golang.org/x/mod v0.21.0
	golang.org/x/net v0.30.0
	golang.org/x/oauth2 v0.23.0
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.26.0
	golang.org/x/term v0.25.0
	golang.org/x/text v0.19.0
	golang.org/x/tools v0.26.0
	google.golang.org/api v0.196.0
	google.golang.org/grpc v1.67.1
	gopkg.in/guregu/null.v3 v3.5.0
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	howett.net/plist v1.0.1
	software.sslmate.com/src/go-pkcs12 v0.4.0
)

require (
	cloud.google.com/go v0.115.1 // indirect
	cloud.google.com/go/auth v0.9.3 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	cloud.google.com/go/iam v1.2.0 // indirect
	cloud.google.com/go/kms v1.19.0 // indirect
	cloud.google.com/go/longrunning v0.6.0 // indirect
	cloud.google.com/go/storage v1.43.0 // indirect
	code.gitea.io/sdk/gitea v0.19.0 // indirect
	dario.cat/mergo v1.0.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/AlekSi/pointer v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.14.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys v0.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.3.2 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/anchore/go-logger v0.0.0-20230725134548-c21dafa1ec5a // indirect
	github.com/anchore/go-macholibre v0.0.0-20220308212642-53e6d0aaf6fb // indirect
	github.com/antchfx/xpath v1.2.2 // indirect
	github.com/apache/thrift v0.18.1 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/atc0005/go-teams-notify/v2 v2.13.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.33 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.32 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.28.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.23.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.35.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.7 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20240514230400-03fa26f5508f // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blacktop/go-dwarf v1.0.10 // indirect
	github.com/blacktop/go-macho v1.1.231 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bluesky-social/indigo v0.0.0-20240813042137-4006c0eca043 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/c-bata/go-prompt v0.2.3 // indirect
	github.com/caarlos0/ctrlc v1.2.0 // indirect
	github.com/caarlos0/env/v11 v11.2.2 // indirect
	github.com/caarlos0/go-reddit/v3 v3.0.1 // indirect
	github.com/caarlos0/go-shellwords v1.0.12 // indirect
	github.com/caarlos0/go-version v0.1.1 // indirect
	github.com/caarlos0/log v0.4.6 // indirect
	github.com/carlmjohnson/versioninfo v0.22.5 // indirect
	github.com/cavaliergopher/cpio v1.0.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/lipgloss v1.0.0 // indirect
	github.com/charmbracelet/x/ansi v0.4.2 // indirect
	github.com/charmbracelet/x/exp/ordered v0.1.0 // indirect
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20230304212654-82a0ddb27589 // indirect
	github.com/cloudflare/circl v1.3.8 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20231011164504-785e29786b46 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davidmz/go-pageant v1.0.2 // indirect
	github.com/dghubble/go-twitter v0.0.0-20211115160449-93a8679adecb // indirect
	github.com/dghubble/oauth1 v0.7.3 // indirect
	github.com/dghubble/sling v1.4.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.1.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.1 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/elastic/go-sysinfo v1.11.2 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/elliotchance/orderedmap/v2 v2.2.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.5 // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-fed/httpsig v1.1.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-restruct/restruct v1.2.0-alpha // indirect
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/go-github/v66 v66.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/ko v0.17.1 // indirect
	github.com/google/rpmpack v0.6.1-0.20240329070804-c2247cbb881a // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/safetext v0.0.0-20220905092116-b49f7bc46da2 // indirect
	github.com/google/wire v0.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.3 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/goreleaser/chglog v0.6.1 // indirect
	github.com/goreleaser/fileglob v1.3.0 // indirect
	github.com/goreleaser/quill v0.0.0-20241025150139-731751b4046d // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.12.0 // indirect
	github.com/ipfs/bbloom v0.0.4 // indirect
	github.com/ipfs/go-block-format v0.2.0 // indirect
	github.com/ipfs/go-cid v0.4.1 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/ipfs/go-ipfs-blockstore v1.3.1 // indirect
	github.com/ipfs/go-ipfs-ds-help v1.1.1 // indirect
	github.com/ipfs/go-ipfs-util v0.0.3 // indirect
	github.com/ipfs/go-ipld-cbor v0.1.0 // indirect
	github.com/ipfs/go-ipld-format v0.6.0 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-log/v2 v2.5.1 // indirect
	github.com/ipfs/go-metrics-interface v0.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/jedisct1/go-minisign v0.0.0-20230811132847-661be99b8267 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kolide/kit v0.0.0-20221107170827-fb85e3d59eab // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/letsencrypt/boulder v0.0.0-20240620165639-de9c06129bec // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-mastodon v0.0.9 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/muesli/mango v0.1.0 // indirect
	github.com/muesli/mango-cobra v1.2.0 // indirect
	github.com/muesli/mango-pflag v0.1.0 // indirect
	github.com/muesli/roff v0.1.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/term v0.0.0-20190109203006-aa71e9d9e942 // indirect
	github.com/polydawn/refmt v0.89.1-0.20221221234430-40501e09de1f // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sassoftware/relic v7.2.1+incompatible // indirect
	github.com/scylladb/go-set v1.0.3-0.20200225121959-cc7b2070d91e // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sigstore/cosign/v2 v2.4.1 // indirect
	github.com/sigstore/protobuf-specs v0.3.2 // indirect
	github.com/sigstore/rekor v1.3.6 // indirect
	github.com/sigstore/sigstore v1.8.9 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.2.2 // indirect
	github.com/slack-go/slack v0.15.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/wagoodman/go-partybus v0.0.0-20230516145632-8ccac152c651 // indirect
	github.com/wagoodman/go-progress v0.0.0-20220614130704-4b1c25a33c7c // indirect
	github.com/whyrusleeping/cbor-gen v0.1.3-0.20240731173018-74d74643234c // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/xanzy/go-gitlab v0.112.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	gitlab.com/digitalxero/go-conventional-commit v1.0.7 // indirect
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.54.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	gocloud.dev v0.40.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/xerrors v0.0.0-20240716161551-93cc26a95ae9 // indirect
	google.golang.org/genproto v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.2.1 // indirect
	sigs.k8s.io/kind v0.24.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
