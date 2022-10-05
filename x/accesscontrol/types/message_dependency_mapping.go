package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	"github.com/gogo/protobuf/proto"
)

var ErrNoCommitAccessOp = fmt.Errorf("MessageDependencyMapping doesn't terminate with AccessType_COMMIT")

type MessageKey string

func GenerateMessageKey(msg sdk.Msg) MessageKey {
	return MessageKey(proto.MessageName(msg))
}

func ValidateMessageDependencyMapping(mapping acltypes.MessageDependencyMapping) error {
	lastAccessOp := mapping.AccessOps[len(mapping.AccessOps)-1]
	if lastAccessOp.AccessType != acltypes.AccessType_COMMIT {
		return ErrNoCommitAccessOp
	}
	return nil
}

func SynchronousMessageDependencyMapping(messageKey MessageKey) acltypes.MessageDependencyMapping {
	return acltypes.MessageDependencyMapping{
		MessageKey: string(messageKey),
		AccessOps: []acltypes.AccessOperation{
			{AccessType: acltypes.AccessType_UNKNOWN, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
			{AccessType: acltypes.AccessType_COMMIT, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
		},
	}
}

func DefaultMessageDependencyMapping() []acltypes.MessageDependencyMapping {
	return []acltypes.MessageDependencyMapping{
		SynchronousMessageDependencyMapping(""),
	}
}
