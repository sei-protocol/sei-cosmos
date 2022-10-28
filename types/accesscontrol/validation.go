package accesscontrol

import (
	"fmt"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	// Param Store Values can only be set during genesis and updated
	// through a gov proposal and those are always processed sequentially
	ConcurrentSafeIdentifiers = map[string]bool{
		"bank/SendEnabled": true,
		"bank/DefaultSendEnabled": true,
	}
)


type Comparator struct {
	AccessType AccessType
	Identifier 	string
}

func (c *Comparator) Contains(comparator Comparator) bool {
	return c.AccessType == comparator.AccessType && strings.Contains(c.Identifier, comparator.Identifier)
}

func (c *Comparator) IsConcurrentSafeIdentifier() bool {
	if val, ok := ConcurrentSafeIdentifiers[c.Identifier]; ok {
		return val
	}
	return false
}

func (c *Comparator) String() string {
	return fmt.Sprintf("AccessType=%s, Identifier=%s\n", c.AccessType, c.Identifier)
}

func AccessTypeStringToEnum(accessType string) AccessType {
	switch strings.ToUpper(accessType) {
	case "WRITE":
		return AccessType_WRITE
	case "READ":
		return AccessType_READ
	default:
		panic(fmt.Sprintf("unknown accessType=%s", accessType))
	}
}

func BuildComparatorFromAccessOp(accessOps []AccessOperation) (comparators []Comparator) {
	for _, accessOp := range accessOps {
		comparators = append(comparators, Comparator{
			AccessType: accessOp.GetAccessType(),
			Identifier: accessOp.GetIdentifierTemplate(),
		})
	}
	return comparators
}

func BuildComparatorFromEvents(events []abci.Event) (comparators []Comparator) {
	for _, event := range events {
		if event.Type != "resource_access" {
			continue
		}
		attributes := event.GetAttributes()

		identifier := ""
		accessType := AccessType_UNKNOWN
		for _, attribute := range attributes {
			if attribute.Key == "key" {
				identifier = attribute.Value
			}
			if attribute.Key == "access_type" {
				accessType =  AccessTypeStringToEnum(attribute.Value)
			}
		}
		comparators = append(comparators, Comparator{
			AccessType: accessType,
			Identifier: identifier,
		})
	}
	return comparators
}

// ValidateAccessOperations compares a list of events and a predefined list of access operations and determines if all the
// events that occurred are represented in the accessOperations
func ValidateAccessOperations(accessOps []AccessOperation, events []abci.Event) map[Comparator]bool {
	eventsComparators := BuildComparatorFromEvents(events)
	accessOpsComparators := BuildComparatorFromAccessOp(accessOps)

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
		for _, accessOpComparator := range accessOpsComparators {
			if  eventComparator.Contains(accessOpComparator){
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
