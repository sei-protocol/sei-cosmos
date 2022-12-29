package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	acltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	"github.com/gogo/protobuf/proto"
)

var (
	ErrNoCommitAccessOp                  = fmt.Errorf("MessageDependencyMapping doesn't terminate with AccessType_COMMIT")
	ErrEmptyIdentifierString             = fmt.Errorf("IdentifierTemplate cannot be an empty string")
	ErrNonLeafResourceTypeWithIdentifier = fmt.Errorf("IdentifierTemplate must be '*' for non leaf resource types")
)

type MessageKey string

func GenerateMessageKey(msg sdk.Msg) MessageKey {
	return MessageKey(proto.MessageName(msg))
}

func CommitAccessOp() *acltypes.AccessOperation {
	return &acltypes.AccessOperation{ResourceType: acltypes.ResourceType_ANY, AccessType: acltypes.AccessType_COMMIT, IdentifierTemplate: "*"}
}

// Validates access operation sequence for a message, requires the last access operation to be a COMMIT
func ValidateAccessOps(accessOps []acltypes.AccessOperation) error {
	lastAccessOp := accessOps[len(accessOps)-1]
	if lastAccessOp != *CommitAccessOp() {
		return ErrNoCommitAccessOp
	}
	for _, accessOp := range accessOps {
		err := ValidateAccessOp(accessOp)
		if err != nil {
			return err
		}
	}

	return nil
}

func ValidateAccessOp(accessOp acltypes.AccessOperation) error {
	if accessOp.IdentifierTemplate == "" {
		return ErrEmptyIdentifierString
	}
	if accessOp.ResourceType.HasChildren() && accessOp.IdentifierTemplate != "*" {
		return ErrNonLeafResourceTypeWithIdentifier
	}
	return nil
}

func ValidateMessageDependencyMapping(mapping acltypes.MessageDependencyMapping) error {
	return ValidateAccessOps(mapping.AccessOps)
}

func SynchronousMessageDependencyMapping(messageKey MessageKey) acltypes.MessageDependencyMapping {
	return acltypes.MessageDependencyMapping{
		MessageKey:     string(messageKey),
		DynamicEnabled: true,
		AccessOps:      acltypes.SynchronousAccessOps(),
	}
}

func SynchronousAccessOps() []acltypes.AccessOperation {
	return []acltypes.AccessOperation{
		{AccessType: acltypes.AccessType_UNKNOWN, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
		*CommitAccessOp(),
	}
}

func SynchronousAccessOpsWithSelector() []acltypes.AccessOperationWithSelector {
	return []acltypes.AccessOperationWithSelector{
		{
			Operation:    &acltypes.AccessOperation{AccessType: acltypes.AccessType_UNKNOWN, ResourceType: acltypes.ResourceType_ANY, IdentifierTemplate: "*"},
			SelectorType: acltypes.AccessOperationSelectorType_NONE,
		},
		{
			Operation:    CommitAccessOp(),
			SelectorType: acltypes.AccessOperationSelectorType_NONE,
		},
	}
}

func SynchronousWasmDependencyMapping(contractAddress string) acltypes.WasmDependencyMapping {
	return acltypes.WasmDependencyMapping{
		Enabled:         true,
		AccessOps:       SynchronousAccessOpsWithSelector(),
		ContractAddress: contractAddress,
	}
}

func IsDefaultSynchronousAccessOps(accessOps []acltypes.AccessOperation) bool {
	defaultAccessOps := SynchronousAccessOps()
	for index, accessOp := range accessOps {
		if accessOp != defaultAccessOps[index] {
			return false
		}
	}
	return true
}

func DefaultMessageDependencyMapping() []acltypes.MessageDependencyMapping {
	return []acltypes.MessageDependencyMapping{
		{
			MessageKey: "cosmos.accesscontrol_x.v1beta1.MsgRegisterWasmDependency",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_ACCESSCONTROL_WASM_DEPENDENCY_MAPPING,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.authz.v1beta1.MsgGrant",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTHZ,
					IdentifierTemplate: "01",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey:     "cosmos.authz.v1beta1.MsgExec",
			DynamicEnabled: false, // could be dispatched to any message type route. Need to add a special selector type for this use case.
		},
		{
			MessageKey: "cosmos.authz.v1beta1.MsgRevoke",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTHZ,
					IdentifierTemplate: "01",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.bank.v1beta1.MsgSend",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.bank.v1beta1.MsgMultiSend",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.crisis.v1beta1.MsgVerifyInvariant",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.distribution.v1beta1.MsgSetWithdrawAddress",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATOR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_STAKING_DELEGATION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_STARTING_INFO,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_CURRENT_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_FEE_POOL,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_HISTORICAL_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_SLASH_EVENT,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_ACCUM_COMMISSION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.distribution.v1beta1.MsgFundCommunityPool",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_FEE_POOL,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey:     "cosmos.evidence.v1beta1.MsgSubmitEvidence",
			DynamicEnabled: false, // could be dispatched to any message type route. Need to add a special selector type for this use case.
		},
		{
			MessageKey: "cosmos.feegrant.v1beta1.MsgGrantAllowance",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_FEEGRANT_ALLOWANCE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.feegrant.v1beta1.MsgRevokeAllowance",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_FEEGRANT_ALLOWANCE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.slashing.v1beta1.MsgUnjail",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATOR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_BY_POWER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_CON_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_SLASHING_VAL_SIGNING_INFO,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.staking.v1beta1.MsgCreateValidator",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATOR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_BY_POWER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_CON_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_ACCUM_COMMISSION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_CURRENT_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_HISTORICAL_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_SLASHING_ADDR_PUBKEY_RELATION_KEY,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_DELEGATION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_STARTING_INFO,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_FEE_POOL,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_SLASH_EVENT,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.staking.v1beta1.MsgEditValidator",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATOR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.staking.v1beta1.MsgDelegate",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATOR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_BY_POWER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_VALIDATORS_CON_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_ACCUM_COMMISSION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_CURRENT_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_VAL_HISTORICAL_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_OUTSTANDING_REWARDS,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_SLASHING_ADDR_PUBKEY_RELATION_KEY,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_STAKING_DELEGATION,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_STARTING_INFO,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_FEE_POOL,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_SLASH_EVENT,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_READ,
					ResourceType:       acltypes.ResourceType_KV_DISTRIBUTION_DELEGATOR_WITHDRAW_ADDR,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
		{
			MessageKey: "cosmos.vesting.v1beta1.MsgCreateVestingAccount",
			AccessOps: []acltypes.AccessOperation{
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_BANK_BALANCES,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_WRITE,
					ResourceType:       acltypes.ResourceType_KV_AUTH_GLOBAL_ACCOUNT_NUMBER,
					IdentifierTemplate: "*",
				},
				{
					AccessType:         acltypes.AccessType_COMMIT,
					ResourceType:       acltypes.ResourceType_ANY,
					IdentifierTemplate: "*",
				},
			},
			DynamicEnabled: true,
		},
	}
}

func DefaultWasmDependencyMappings() []acltypes.WasmDependencyMapping {
	return []acltypes.WasmDependencyMapping{}
}

func ValidateWasmDependencyMapping(mapping acltypes.WasmDependencyMapping) error {
	lastAccessOp := mapping.AccessOps[len(mapping.AccessOps)-1]
	if lastAccessOp.Operation.AccessType != acltypes.AccessType_COMMIT {
		return ErrNoCommitAccessOp
	}
	return nil
}
