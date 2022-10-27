package accesscontrol

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	EmptyPrefix = []byte{}
	ParentNodeKey = "ParentNode"
)

type StoreKeyToResourceTypePrefixMap map[string]map[ResourceType][]byte;

func DefaultStoreKeyToResourceTypePrefixMap() StoreKeyToResourceTypePrefixMap {
	return StoreKeyToResourceTypePrefixMap{
		ParentNodeKey: {
			ResourceType_ANY:     EmptyPrefix,
			ResourceType_KV:      EmptyPrefix,
			ResourceType_Mem:     EmptyPrefix,
			ResourceType_KV_WASM: EmptyPrefix,
		},
	}
}

type MsgValidator struct {
	storeKeyToResourceTypePrefixMap StoreKeyToResourceTypePrefixMap
}

func NewMsgValidator(storeKeyToResourceTypePrefixMap StoreKeyToResourceTypePrefixMap) *MsgValidator {
	return &MsgValidator{
		storeKeyToResourceTypePrefixMap: storeKeyToResourceTypePrefixMap,
	}
}

// ValidateAccessOperations compares a list of events and a predefined list of access operations and determines if all the
// events that occurred are represented in the accessOperations
func (validator *MsgValidator) ValidateAccessOperations(accessOps []AccessOperation, events []abci.Event) map[Comparator]bool {
	eventsComparators := BuildComparatorFromEvents(events, validator.storeKeyToResourceTypePrefixMap)
	missingAccessOps := make(map[Comparator]bool)

	// If it's using default synchronous access op mapping then no need to verify
	if IsDefaultSynchronousAccessOps(accessOps) {
		return missingAccessOps
	}

	for _, eventComparator := range eventsComparators {
		if eventComparator.IsConcurrentSafeIdentifier() {
			continue
		}

		matched := false
		for _, accessOps := range accessOps {
			if  eventComparator.DependencyMatch(accessOps) {
				matched = true
				break
			}
		}

		if !matched {
			missingAccessOps[eventComparator] = true
		}

	}
	return missingAccessOps
}
