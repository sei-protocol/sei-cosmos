go 1.21

module github.com/cosmos/cosmos-sdk

require (
	cosmossdk.io/errors v1.0.0
	github.com/99designs/keyring v1.2.1
	github.com/armon/go-metrics v0.4.1
	github.com/bgentry/speakeasy v0.1.0
	github.com/btcsuite/btcd v0.22.1
	github.com/coinbase/rosetta-sdk-go v0.7.0
	github.com/confio/ics23/go v0.9.0
	github.com/cosmos/btcutil v1.0.5
	github.com/cosmos/go-bip39 v1.0.0
	github.com/cosmos/iavl v0.21.0-alpha.1.0.20230904092046-df3db2d96583
	github.com/cosmos/ledger-cosmos-go v0.12.2
	github.com/deckarep/golang-set v1.8.0
	github.com/ethereum/go-ethereum v1.13.2
	github.com/gogo/gateway v1.1.0
	github.com/gogo/protobuf v1.3.3
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/google/btree v1.1.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru/v2 v2.0.1
	github.com/hdevalence/ed25519consensus v0.0.0-20220222234857-c00d1f31bab3
	github.com/improbable-eng/grpc-web v0.14.1
	github.com/jhump/protoreflect v1.12.1-0.20220417024638-438db461d753
	github.com/magiconair/properties v1.8.6
	github.com/mattn/go-isatty v0.0.19
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/common v0.42.0
	github.com/rakyll/statik v0.1.7
	github.com/regen-network/cosmos-proto v0.3.1
	github.com/rs/zerolog v1.30.0
	github.com/savaki/jq v0.0.0-20161209013833-0e6baecebbf8
	github.com/sei-protocol/sei-db v0.0.27-0.20240123064153-d6dfa112e760
	github.com/sei-protocol/sei-tm-db v0.0.5
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.4
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
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
	golang.org/x/crypto v0.15.0
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa
	google.golang.org/genproto/googleapis/api v0.0.0-20230920204549-e6e6cdab5c13
	google.golang.org/grpc v1.58.3
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

require (
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/dvsekhvalnov/jose2go v1.5.0 // indirect
	github.com/gin-gonic/gin v1.7.7 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231012201019-e917dd12ba7a // indirect
)

replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0
	github.com/cosmos/iavl => github.com/sei-protocol/sei-iavl v0.1.9
	github.com/ethereum/go-ethereum => github.com/sei-protocol/go-ethereum v1.13.5-sei-8
	// Fix upstream GHSA-h395-qcrw-5vmq vulnerability.
	// TODO Remove it: https://github.com/cosmos/cosmos-sdk/issues/10409
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.0
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/sei-protocol/sei-db => github.com/sei-protocol/sei-db v0.0.31
	// Latest goleveldb is broken, we have to stick to this version
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/tendermint/tendermint => github.com/sei-protocol/sei-tendermint v0.2.38-evm-rebase-2
	// latest grpc doesn't work with with our modified proto compiler, so we need to enforce
	// the following version across all dependencies.
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
