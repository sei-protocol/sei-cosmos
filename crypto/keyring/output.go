package keyring

import (
	"encoding/hex"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/hd"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// TODO: Move this file to client/keys
// Use protobuf interface marshaler rather then generic JSON

// KeyOutput defines a structure wrapping around an Info object used for output
// functionality.
type KeyOutput struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"`
	Address    string `json:"address" yaml:"address"`
	EvmAddress string `json:"evm_address" yaml:"evm_address"`
	PubKey     string `json:"pubkey" yaml:"pubkey"`
	Mnemonic   string `json:"mnemonic,omitempty" yaml:"mnemonic"`
}

// NewKeyOutput creates a default KeyOutput instance without Mnemonic, Threshold and PubKeys
func NewKeyOutput(name string, keyType KeyType, a sdk.Address, pk cryptotypes.PubKey) (KeyOutput, error) { // nolint:interfacer
	apk, err := codectypes.NewAnyWithValue(pk)
	if err != nil {
		return KeyOutput{}, err
	}
	bz, err := codec.ProtoMarshalJSON(apk, nil)
	if err != nil {
		return KeyOutput{}, err
	}
	return KeyOutput{
		Name:    name,
		Type:    keyType.String(),
		Address: a.String(),
		PubKey:  string(bz),
	}, nil
}

// MkConsKeyOutput create a KeyOutput in with "cons" Bech32 prefixes.
func MkConsKeyOutput(keyInfo Info) (KeyOutput, error) {
	pk := keyInfo.GetPubKey()
	addr := sdk.ConsAddress(pk.Address())
	return NewKeyOutput(keyInfo.GetName(), keyInfo.GetType(), addr, pk)
}

// MkValKeyOutput create a KeyOutput in with "val" Bech32 prefixes.
func MkValKeyOutput(keyInfo Info) (KeyOutput, error) {
	pk := keyInfo.GetPubKey()
	addr := sdk.ValAddress(pk.Address())
	return NewKeyOutput(keyInfo.GetName(), keyInfo.GetType(), addr, pk)
}

// MkAccKeyOutput create a KeyOutput in with "acc" Bech32 prefixes. If the
// public key is a multisig public key, then the threshold and constituent
// public keys will be added.
func MkAccKeyOutput(keyInfo Info) (KeyOutput, error) {
	pk := keyInfo.GetPubKey()
	addr := sdk.AccAddress(pk.Address())
	return NewKeyOutput(keyInfo.GetName(), keyInfo.GetType(), addr, pk)
}

// MkAccKeysOutput returns a slice of KeyOutput objects, each with the "acc"
// Bech32 prefixes, given a slice of Info objects. It returns an error if any
// call to MkKeyOutput fails.
func MkAccKeysOutput(infos []Info) ([]KeyOutput, error) {
	kos := make([]KeyOutput, len(infos))
	var err error
	for i, info := range infos {
		kos[i], err = MkAccKeyOutput(info)
		fmt.Printf("PSUDEBUG - finished MkAccKeyOutputM %s\n", info)
		if err != nil {
			fmt.Println("PSUDEBUG - err MkAccKeyOutputM: %s\n", err)
			return nil, err
		}
		if info.GetAlgo() == hd.Secp256k1Type {
			kos[i], err = PopulateEvmAddrIfApplicable(info, kos[i])
			fmt.Printf("PSUDEBUG - finished PopulateEvmAddrIfApplicable\n")
			if err != nil {
				fmt.Printf("PSUDEBUG - err PopulateEvmAddrIfApplicable: %s\n", err)
				return nil, err
			}
		}
	}

	return kos, nil
}

func PopulateEvmAddrIfApplicable(info Info, o KeyOutput) (KeyOutput, error) {
	fmt.Printf("PSUDEBUG - starting cast to LocalInfo\n")
	localInfo, ok := info.(LocalInfo)
	fmt.Printf("PSUDEBUG - finished cast to LocalInfo %s\n", ok)
	if ok {
		priv, err := legacy.PrivKeyFromBytes([]byte(localInfo.PrivKeyArmor))
		if err != nil {
			fmt.Printf("PSUDEBUG - PrivKeyFromBytes err : %s\n", err)
			return o, err
		}
		privHex := hex.EncodeToString(priv.Bytes())
		privKey, err := crypto.HexToECDSA(privHex)
		if err != nil {
			fmt.Printf("PSUDEBUG - HexToECDSA err : %s\n", err)
			return o, err
		}
		o.EvmAddress = crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	} else {
		fmt.Printf("PSUDEBUG - LocalInfo err\n")
	}
	return o, nil
}
