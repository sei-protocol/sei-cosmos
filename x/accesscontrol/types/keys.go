package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

// ResourceDependencyMappingKey is the key used for the keeper store
var (
	ResourceDependencyMappingKey = 0x01
	WasmMappingKey               = 0x02
)

const (
	// ModuleName defines the module name
	ModuleName = "accesscontrol"

	QuerierRoute = ModuleName

	// Append "acl" to prevent prefix collision with "acc" module
	StoreKey = "acl" + ModuleName

	RouterKey = ModuleName
)

func GetResourceDependencyMappingKey() []byte {
	return []byte{byte(ResourceDependencyMappingKey)}
}

func GetResourceDependencyKey(messageKey MessageKey) []byte {
	return append(GetResourceDependencyMappingKey(), []byte(messageKey)...)
}

func GetWasmMappingKey() []byte {
	return []byte{byte(WasmMappingKey)}
}

func GetWasmContractAddressPrefix(contractAddress sdk.AccAddress) []byte {
	return append(GetWasmMappingKey(), address.MustLengthPrefix(contractAddress)...)
}

// wasmFunctionName is the top level object key in the execute JSON payload
func GetWasmFunctionDependencyKey(contractAddress sdk.AccAddress, wasmFunctionName string) []byte {
	return append(GetWasmContractAddressPrefix(contractAddress), []byte(wasmFunctionName)...)
}
