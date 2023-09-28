go 1.18

module github.com/cosmos/cosmos-sdk

require (
	cosmossdk.io/log v1.2.1
	github.com/99designs/keyring v1.1.6
	github.com/armon/go-metrics v0.3.10
	github.com/bgentry/speakeasy v0.1.0
	github.com/btcsuite/btcd v0.22.1
	github.com/coinbase/rosetta-sdk-go v0.7.0
	github.com/confio/ics23/go v0.7.0
	github.com/cosmos/btcutil v1.0.4
	github.com/cosmos/go-bip39 v1.0.0
	github.com/cosmos/iavl v0.19.4
	github.com/cosmos/ledger-cosmos-go v0.11.1
	github.com/deckarep/golang-set v1.8.0
	github.com/gogo/gateway v1.1.0
	github.com/gogo/protobuf v1.3.3
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru/v2 v2.0.1
	github.com/hdevalence/ed25519consensus v0.0.0-20210204194344-59a8610d2b87
	github.com/improbable-eng/grpc-web v0.14.1
	github.com/jhump/protoreflect v1.12.1-0.20220417024638-438db461d753
	github.com/magiconair/properties v1.8.6
	github.com/mattn/go-isatty v0.0.19
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/common v0.34.0
	github.com/rakyll/statik v0.1.7
	github.com/regen-network/cosmos-proto v0.3.1
	github.com/rs/zerolog v1.30.0
	github.com/savaki/jq v0.0.0-20161209013833-0e6baecebbf8
	github.com/sei-protocol/sei-tm-db v0.0.5
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tendermint/btcd v0.1.1
	github.com/tendermint/crypto v0.0.0-20191022145703-50d29ede1e15
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.37.0-dev
	github.com/tendermint/tm-db v0.6.8-0.20220519162814-e24b96538a12
	github.com/yourbasic/graph v0.0.0-20210606180040-8ecfec1c2869
	go.opentelemetry.io/otel v1.9.0
	go.opentelemetry.io/otel/exporters/jaeger v1.9.0
	go.opentelemetry.io/otel/sdk v1.9.0
	go.opentelemetry.io/otel/trace v1.9.0
	golang.org/x/crypto v0.1.0
	golang.org/x/exp v0.0.0-20221026153819-32f3d567a233
	google.golang.org/genproto v0.0.0-20221014213838-99cd37c6964a
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

require (
	github.com/dustin/go-humanize v1.0.1-0.20200219035652-afde56e7acac // indirect
	github.com/gin-gonic/gin v1.7.7 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0

	github.com/cosmos/iavl => github.com/sei-protocol/sei-iavl v0.1.7

	// Fix upstream GHSA-h395-qcrw-5vmq vulnerability.
	// TODO Remove it: https://github.com/cosmos/cosmos-sdk/issues/10409
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

	github.com/tendermint/tendermint => github.com/sei-protocol/sei-tendermint v0.2.25

	// latest grpc doesn't work with with our modified proto compiler, so we need to enforce
	// the following version across all dependencies.
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
