package accesscontrol

import (
	fmt "fmt"
)

func SynchronousAccessOps() []AccessOperation {
	return []AccessOperation{
		{AccessType: AccessType_UNKNOWN, ResourceType: ResourceType_ANY, IdentifierTemplate: "*"},
		{AccessType: AccessType_COMMIT, ResourceType: ResourceType_ANY, IdentifierTemplate: "*"},
	}
}

func (a *AccessOperation) GetResourceIDTemplate(args []any) string {
	return fmt.Sprintf(a.GetIdentifierTemplate(), args...)
}

func IsDefaultSynchronousAccessOps(accessOps []AccessOperation) bool {
	defaultAccessOps := SynchronousAccessOps()
	for index, accessOp := range accessOps {
		if accessOp != defaultAccessOps[index] {
			return false
		}
	}
	return true
}
