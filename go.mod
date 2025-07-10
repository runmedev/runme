module github.com/runmedev/runme/v3

go 1.24

toolchain go1.24.2

// replace github.com/stateful/godotenv => ../godotenv

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.6-20250625184727-c923a0c2a132.1
	buf.build/go/protovalidate v0.13.1
	cloud.google.com/go/secretmanager v1.15.0
	connectrpc.com/connect v1.18.1
	connectrpc.com/cors v0.1.0
	connectrpc.com/grpchealth v1.4.0
	connectrpc.com/otelconnect v0.7.2
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/Microsoft/go-winio v0.6.2
	github.com/atotto/clipboard v0.1.4
	github.com/charmbracelet/bubbletea v1.3.6
	github.com/charmbracelet/lipgloss v1.1.1-0.20250319133953-166f707985bc
	github.com/cli/cli/v2 v2.74.2
	github.com/cli/go-gh/v2 v2.12.1
	github.com/containerd/console v1.0.5
	github.com/creack/pty v1.1.24
	github.com/docker/docker v28.3.1+incompatible
	github.com/expr-lang/expr v1.17.5
	github.com/fatih/color v1.18.0
	github.com/fullstorydev/grpcurl v1.9.3
	github.com/gabriel-vasile/mimetype v1.4.9
	github.com/go-git/go-billy/v5 v5.6.2
	github.com/go-logr/logr v1.4.3
	github.com/go-logr/zapr v1.3.0
	github.com/gobwas/glob v0.2.3
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/golang/mock v1.7.0-rc.1
	github.com/google/go-github/v45 v45.2.0
	github.com/gorilla/websocket v1.5.3
	github.com/graphql-go/graphql v0.8.1
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jhump/protoreflect v1.17.0
	github.com/jlewi/monogo v0.0.0-20241216141120-2e83e825aa81
	github.com/mattn/go-isatty v0.0.20
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d
	github.com/muesli/cancelreader v0.2.2
	github.com/oklog/ulid/v2 v2.1.1
	github.com/openai/openai-go v1.0.0
	github.com/opencontainers/image-spec v1.1.1
	github.com/otiai10/copy v1.14.1
	github.com/rogpeppe/go-internal v1.14.1
	github.com/rs/cors v1.11.1
	github.com/spf13/viper v1.20.1
	github.com/stateful/godotenv v0.0.0-20240309032207-c7bc0b812915
	github.com/vektah/gqlparser/v2 v2.5.30
	github.com/xo/dburl v0.23.8
	github.com/yuin/goldmark v1.7.12
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0
	go.opentelemetry.io/otel/sdk v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	go.uber.org/dig v1.19.0
	go.uber.org/multierr v1.11.0
	golang.org/x/net v0.41.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/sys v0.33.0
	golang.org/x/term v0.32.0
	google.golang.org/api v0.240.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7
	google.golang.org/protobuf v1.36.6
	mvdan.cc/sh/v3 v3.12.0
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.121.3 // indirect
	cloud.google.com/go/ai v0.8.0 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/logging v1.13.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/atombender/go-jsonschema v0.16.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/bufbuild/protocompile v0.14.1 // indirect
	github.com/ccojocar/zxcvbn-go v1.0.2 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/charmbracelet/colorprofile v0.3.1 // indirect
	github.com/charmbracelet/x/ansi v0.9.3 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/chavacava/garif v0.1.0 // indirect
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-cmd/cmd v1.4.3 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/goccy/go-yaml v1.11.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/cel-go v0.25.0 // indirect
	github.com/google/generative-ai-go v0.19.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/icholy/gomajor v0.13.1 // indirect
	github.com/mgechev/dots v0.0.0-20210922191527-e955255bf517 // indirect
	github.com/mgechev/revive v1.7.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/otiai10/mint v1.6.3 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/securego/gosec/v2 v2.22.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	gotest.tools/v3 v3.5.1 // indirect
	gvisor.dev/gvisor v0.0.0-20250318191406-9e676ea1de20 // indirect
	honnef.co/go/tools v0.6.0 // indirect
	mvdan.cc/gofumpt v0.7.0 // indirect
)

require (
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/briandowns/spinner v1.23.2 // indirect
	github.com/cli/go-gh v1.2.1
	github.com/cli/safeexec v1.0.1 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/protobuf v1.5.4
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Khan/genqlient v0.8.1
	github.com/charmbracelet/bubbles v0.21.0
	github.com/elliotchance/orderedmap v1.8.0
	github.com/go-git/go-git/v5 v5.16.2
	github.com/go-playground/assert/v2 v2.2.0
	github.com/go-playground/validator/v10 v10.27.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/henvic/httpretty v0.1.4
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v1.2.0-beta.2.0.20211217091447-1a4a3b719465
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.15.0
	google.golang.org/grpc v1.73.0
)

tool (
	github.com/atombender/go-jsonschema
	github.com/icholy/gomajor
	github.com/mgechev/revive
	github.com/securego/gosec/v2/cmd/gosec
	golang.org/x/tools/cmd/goimports
	gvisor.dev/gvisor/tools/checklocks/cmd/checklocks
	honnef.co/go/tools/cmd/staticcheck
	mvdan.cc/gofumpt
)
