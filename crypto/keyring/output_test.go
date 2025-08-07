package keyring

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/sr25519"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestBech32KeysOutput(t *testing.T) {
	sk := secp256k1.PrivKey{Key: []byte{154, 49, 3, 117, 55, 232, 249, 20, 205, 216, 102, 7, 136, 72, 177, 2, 131, 202, 234, 81, 31, 208, 46, 244, 179, 192, 167, 163, 142, 117, 246, 13}}
	tmpKey := sk.PubKey()
	multisigPk := kmultisig.NewLegacyAminoPubKey(1, []types.PubKey{tmpKey})

	info, err := NewMultiInfo("multisig", multisigPk)
	require.NoError(t, err)
	accAddr := sdk.AccAddress(info.GetPubKey().Address())
	expectedOutput, err := NewKeyOutput(info.GetName(), info.GetType(), accAddr, multisigPk)
	require.NoError(t, err)

	out, err := MkAccKeyOutput(info)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, out)
	require.Equal(t, `{Name:multisig Type:multi Address:cosmos1nf8lf6n4wa43rzmdzwe6hkrnw5guekhqt595cw EvmAddress: PubKey:{"@type":"/cosmos.crypto.multisig.LegacyAminoPubKey","threshold":1,"public_keys":[{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"AurroA7jvfPd1AadmmOvWM2rJSwipXfRf8yD6pLbA2DJ"}]} Mnemonic:}`, fmt.Sprintf("%+v", out))
}

func TestMkAccKeyOutputForSr25519(t *testing.T) {
	sk := sr25519.GenPrivKey()
	tmpKey := sk.PubKey()
	multisigPk := kmultisig.NewLegacyAminoPubKey(1, []types.PubKey{tmpKey})

	info, err := NewMultiInfo("multisig", multisigPk)
	require.NoError(t, err)
	accAddr := sdk.AccAddress(info.GetPubKey().Address())
	expectedOutput, err := NewKeyOutput(info.GetName(), info.GetType(), accAddr, multisigPk)
	require.NoError(t, err)

	out, err := MkAccKeyOutput(info)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, out)
}

func TestPopulateEvmAddrIfApplicable(t *testing.T) {
	sk := secp256k1.GenPrivKey()
	pubKey := sk.PubKey()

	// PrivKeyArmor should contain amino-encoded private key bytes
	aminoBytes, err := legacy.Cdc.Marshal(sk)
	require.NoError(t, err)
	privKeyArmor := string(aminoBytes)

	tests := []struct {
		name        string
		info        Info
		input       KeyOutput
		expectError bool
		expectEvm   bool
	}{
		{
			name: "LocalInfo pointer case - should populate EVM address",
			info: &LocalInfo{
				Name:         "test-key",
				PubKey:       pubKey,
				PrivKeyArmor: privKeyArmor,
				Algo:         hd.Secp256k1Type,
			},
			input: KeyOutput{
				Name:    "test-key",
				Type:    "local",
				Address: sdk.AccAddress(pubKey.Address()).String(),
				PubKey:  "",
			},
			expectError: false,
			expectEvm:   true,
		},
		{
			name: "LocalInfo value case - should populate EVM address",
			info: LocalInfo{
				Name:         "test-key",
				PubKey:       pubKey,
				PrivKeyArmor: privKeyArmor,
				Algo:         hd.Secp256k1Type,
			},
			input: KeyOutput{
				Name:    "test-key",
				Type:    "local",
				Address: sdk.AccAddress(pubKey.Address()).String(),
				PubKey:  "",
			},
			expectError: false,
			expectEvm:   true,
		},
		{
			name: "Non-LocalInfo case - should return unchanged",
			info: &multiInfo{
				Name:      "multi-key",
				PubKey:    pubKey,
				Threshold: 1,
			},
			input: KeyOutput{
				Name:       "multi-key",
				Type:       "multi",
				Address:    sdk.AccAddress(pubKey.Address()).String(),
				EvmAddress: "0x1234567890123456789012345678901234567890",
				PubKey:     "",
			},
			expectError: false,
			expectEvm:   false,
		},
		{
			name: "Invalid private key armor - should return error",
			info: &LocalInfo{
				Name:         "bad-key",
				PubKey:       pubKey,
				PrivKeyArmor: "invalid-armor",
				Algo:         hd.Secp256k1Type,
			},
			input: KeyOutput{
				Name:    "bad-key",
				Type:    "local",
				Address: sdk.AccAddress(pubKey.Address()).String(),
				PubKey:  "",
			},
			expectError: true,
			expectEvm:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PopulateEvmAddrIfApplicable(tt.info, tt.input)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.input.Name, result.Name)
			require.Equal(t, tt.input.Type, result.Type)
			require.Equal(t, tt.input.Address, result.Address)

			if tt.expectEvm {
				require.NotEmpty(t, result.EvmAddress)
				require.Len(t, result.EvmAddress, 42) // 0x + 40 hex chars
				require.True(t, result.EvmAddress[:2] == "0x")
			} else {
				require.Equal(t, tt.input.EvmAddress, result.EvmAddress)
			}
		})
	}
}
