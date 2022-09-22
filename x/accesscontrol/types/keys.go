package types

// ResourceDependencyMappingKey is the key used for the keeper store
var ResourceDependencyMappingKey = 0x01

const (
	// ModuleName defines the module name
	ModuleName = "accesscontrol"

	QuerierRoute = ModuleName

	StoreKey = ModuleName
)

func GetResourceDependencyMappingKey() []byte {
	return []byte{byte(ResourceDependencyMappingKey)}
}

func GetResourceKey(moduleName string, messageType string) []byte {
	tempKey := append(GetResourceDependencyMappingKey(), []byte(moduleName)...)
	return append(tempKey, []byte(messageType)...)
}
