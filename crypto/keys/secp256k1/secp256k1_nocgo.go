//go:build !libsecp256k1_sdk
// +build !libsecp256k1_sdk

package secp256k1

import (
	"crypto/sha256"
	"math/big"

	secp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto/utils"
)

// used to reject malleable signatures
// see:
//   - https://github.com/ethereum/go-ethereum/blob/f9401ae011ddf7f8d2d95020b7446c17f8d98dc1/crypto/signature_nocgo.go#L90-L93
//   - https://github.com/ethereum/go-ethereum/blob/f9401ae011ddf7f8d2d95020b7446c17f8d98dc1/crypto/crypto.go#L39
var secp256k1halfN = new(big.Int).Rsh(secp256k1.S256().N, 1)

// Sign creates an ECDSA signature on curve Secp256k1, using SHA256 on the msg.
// The returned signature will be of the form R || S (in lower-S form).
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
	priv, pub := secp256k1.PrivKeyFromBytes(privKey.Key)
	seed := sha256.Sum256(cosmoscrypto.Sha256(msg))
	sigBytes, err := ecdsa.SignCompact(priv, seed[:], len(pub.SerializeCompressed()) == 33)
	if err != nil {
		return nil, err
	}
	return sigBytes[1:], nil
}

// VerifySignature verifies a signature of the form R || S.
// It uses the standard btcec/v2 signature verification approach.
func (pubKey *PubKey) VerifySignature(msg []byte, sigStr []byte) bool {
	if len(sigStr) != 64 {
		return false
	}
	p, err := secp256k1.ParsePubKey(pubKey.Key)
	if err != nil {
		return false
	}

	// Construct signature from R || S bytes using the standard approach
	r := new(big.Int).SetBytes(sigStr[:32])
	s := new(big.Int).SetBytes(sigStr[32:64])

	var rScalar, sScalar secp256k1.ModNScalar
	rScalar.SetByteSlice(r.Bytes())
	sScalar.SetByteSlice(s.Bytes())
	signature := ecdsa.NewSignature(&rScalar, &sScalar)

	// Use standard signature verification
	seed := sha256.Sum256(cosmoscrypto.Sha256(msg))
	return signature.Verify(seed[:], p)
}
