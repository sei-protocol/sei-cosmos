package ledger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestErrorHandling(t *testing.T) {
	// first, try to generate a key, must return an error
	// (no panic)
	path := *hd.NewParams(44, 555, 0, false, 0)
	_, err := NewPrivKeySecp256k1Unsafe(path)
	require.Error(t, err)
}

func TestPublicKeyUnsafe(t *testing.T) {
	path := *hd.NewFundraiserParams(0, sdk.GetConfig().GetCoinType(), 0)
	priv, err := NewPrivKeySecp256k1Unsafe(path)
	require.NoError(t, err)
	checkDefaultPubKey(t, priv)
}

func checkDefaultPubKey(t *testing.T, priv types.LedgerPrivKey) {
	require.NotNil(t, priv)
	expectedPkStr := "PubKeySecp256k1{021853D93524119EEB31AB0B06F1DCB068F84943BB230DFA10B1292F47AF643575}"
	require.Equal(t, "eb5ae98721021853d93524119eeb31ab0b06f1dcb068f84943bb230dfa10b1292f47af643575",
		fmt.Sprintf("%x", cdc.Amino.MustMarshalBinaryBare(priv.PubKey())),
		"Is your device using test mnemonic: %s ?", testdata.TestMnemonic)
	require.Equal(t, expectedPkStr, priv.PubKey().String())
	addr := sdk.AccAddress(priv.PubKey().Address()).String()
	require.Equal(t, "cosmos1ndgvlye77dmuct2ncr4m8ghmfkwp7p8slz6zxm",
		addr, "Is your device using test mnemonic: %s ?", testdata.TestMnemonic)
}

func TestPublicKeyUnsafeHDPath(t *testing.T) {
	expectedAnswers := []string{
		"PubKeySecp256k1{021853D93524119EEB31AB0B06F1DCB068F84943BB230DFA10B1292F47AF643575}",
		"PubKeySecp256k1{022374F2DACD71042B5A888E3839E4BA54752AD6A51D35B54F6ABB899C4329D4BF}",
		"PubKeySecp256k1{03B7B924449C6C983CCA4CF01D7AC53811874C384D1EB18E04916EE9486AE89023}",
		"PubKeySecp256k1{037795D6F3A390A2B83838DEE4F7DDBCE7FC9B745516B1F3031D858BCC5044B0A3}",
		"PubKeySecp256k1{033729F993AE4D3347454A0D37D13C6E3484E7DC7DE7A63DDB4BCA556CCA3886E1}",
		"PubKeySecp256k1{025FEB9BB0D950EFACAEBC163F17E76FAA4984B3FF3D41DD06DB79B9EEBBD930F9}",
		"PubKeySecp256k1{02E59347ED32D311543F1E82B4467684DC4995EEC4700437386E668D3D9E14C1E0}",
		"PubKeySecp256k1{039D92D20650624E2C4A8CAE0421154FC59F885E9518D337BC2105DF8BF13E151B}",
		"PubKeySecp256k1{03BE014D2A23A2D46ED95201DDA9D685DA0086B1A2978E3AC770E03E1C19A778F3}",
		"PubKeySecp256k1{0325E57703629DB24D61697147674FCB10B94ACA0136A89D37C00FC5CFD31B443A}",
	}

	const numIters = 10

	privKeys := make([]types.LedgerPrivKey, numIters)

	// Check with device
	for i := uint32(0); i < 10; i++ {
		path := *hd.NewFundraiserParams(0, sdk.GetConfig().GetCoinType(), i)
		t.Logf("Checking keys at %v\n", path)

		priv, err := NewPrivKeySecp256k1Unsafe(path)
		require.NoError(t, err)
		require.NotNil(t, priv)

		// Check other methods
		tmp := priv.(PrivKeyLedgerSecp256k1)
		require.NoError(t, tmp.ValidateKey())
		(&tmp).AssertIsPrivKeyInner()

		// in this test we are chekcking if the generated keys are correct.
		require.Equal(t, expectedAnswers[i], priv.PubKey().String(),
			"Is your device using test mnemonic: %s ?", testdata.TestMnemonic)

		// Store and restore
		serializedPk := priv.Bytes()
		require.NotNil(t, serializedPk)
		require.True(t, len(serializedPk) >= 50)

		privKeys[i] = priv
	}

	// Now check equality
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			require.Equal(t, i == j, privKeys[i].Equals(privKeys[j]))
			require.Equal(t, i == j, privKeys[j].Equals(privKeys[i]))
		}
	}
}

func TestPublicKeySafe(t *testing.T) {
	path := *hd.NewFundraiserParams(0, sdk.GetConfig().GetCoinType(), 0)
	priv, addr, err := NewPrivKeySecp256k1(path, "cosmos")

	require.NoError(t, err)
	require.NotNil(t, priv)
	require.Nil(t, ShowAddress(path, priv.PubKey(), sdk.GetConfig().GetBech32AccountAddrPrefix()))
	checkDefaultPubKey(t, priv)

	addr2 := sdk.AccAddress(priv.PubKey().Address()).String()
	require.Equal(t, addr, addr2)
}

func TestPublicKeyHDPath(t *testing.T) {
	expectedPubKeys := []string{
		"PubKeySecp256k1{021853D93524119EEB31AB0B06F1DCB068F84943BB230DFA10B1292F47AF643575}",
		"PubKeySecp256k1{022374F2DACD71042B5A888E3839E4BA54752AD6A51D35B54F6ABB899C4329D4BF}",
		"PubKeySecp256k1{03B7B924449C6C983CCA4CF01D7AC53811874C384D1EB18E04916EE9486AE89023}",
		"PubKeySecp256k1{037795D6F3A390A2B83838DEE4F7DDBCE7FC9B745516B1F3031D858BCC5044B0A3}",
		"PubKeySecp256k1{033729F993AE4D3347454A0D37D13C6E3484E7DC7DE7A63DDB4BCA556CCA3886E1}",
		"PubKeySecp256k1{025FEB9BB0D950EFACAEBC163F17E76FAA4984B3FF3D41DD06DB79B9EEBBD930F9}",
		"PubKeySecp256k1{0377A1C826D3A03CA4EE94FC4DEA6BCCB2BAC5F2AC0419A128C29F8E88F1FF295A}",
		"PubKeySecp256k1{039D92D20650624E2C4A8CAE0421154FC59F885E9518D337BC2105DF8BF13E151B}",
		"PubKeySecp256k1{038905A42433B1D677CC8AFD36861430B9A8529171B0616F733659F131C3F80221}",
		"PubKeySecp256k1{038BE7F348902D8C20BC88D32294F4F3B819284548122229DECD1ADF1A7EB0848B}",
	}

	expectedAddrs := []string{
		"cosmos1ndgvlye77dmuct2ncr4m8ghmfkwp7p8slz6zxm",
		"cosmos148thdqj6vnkkmsfd58ej4xjuacmqq7qwawg0ak",
		"cosmos1dj36dxe0sgxygfya5ghggvrhe92qqwm2xcqhtu",
		"cosmos1zhjvvpr8skduawtkz2gqdvpe3w8jj60z637arr",
		"cosmos1r3amgzyuktpfzez9gyhj5dnpwm207gc32zcejc",
		"cosmos146hlrlypx59e5sl50cfys75zcw6s5mymnqm4dj",
		"cosmos1cwus47jk96mpc9t8aczzthwq2eshsfygxhrjlj",
		"cosmos1cq83av8cmnar79h0rg7duh9gnr7wkh228a7fxg",
		"cosmos1dszhfrt226jy5rsre7e48vw9tgwe90uerfyefa",
		"cosmos1734d7qsylzrdt05muhqqtpd90j8mp4y6rzch8l",
	}

	const numIters = 10

	privKeys := make([]types.LedgerPrivKey, numIters)

	// Check with device
	for i := 0; i < len(expectedAddrs); i++ {
		path := *hd.NewFundraiserParams(0, sdk.GetConfig().GetCoinType(), uint32(i))
		t.Logf("Checking keys at %s\n", path)

		priv, addr, err := NewPrivKeySecp256k1(path, "cosmos")
		require.NoError(t, err)
		require.NotNil(t, addr)
		require.NotNil(t, priv)

		addr2 := sdk.AccAddress(priv.PubKey().Address()).String()
		require.Equal(t, addr2, addr)
		require.Equal(t,
			expectedAddrs[i], addr,
			"Is your device using test mnemonic: %s ?", testdata.TestMnemonic)

		// Check other methods
		tmp := priv.(PrivKeyLedgerSecp256k1)
		require.NoError(t, tmp.ValidateKey())
		(&tmp).AssertIsPrivKeyInner()

		// in this test we are chekcking if the generated keys are correct and stored in a right path.
		require.Equal(t,
			expectedPubKeys[i], priv.PubKey().String(),
			"Is your device using test mnemonic: %s ?", testdata.TestMnemonic)

		// Store and restore
		serializedPk := priv.Bytes()
		require.NotNil(t, serializedPk)
		require.True(t, len(serializedPk) >= 50)

		privKeys[i] = priv
	}

	// Now check equality
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			require.Equal(t, i == j, privKeys[i].Equals(privKeys[j]))
			require.Equal(t, i == j, privKeys[j].Equals(privKeys[i]))
		}
	}
}

func getFakeTx(accountNumber uint32) []byte {
	tmp := fmt.Sprintf(
		`{"account_number":"%d","chain_id":"1234","fee":{"amount":[{"amount":"150","denom":"atom"}],"gas":"5000"},"memo":"memo","msgs":[[""]],"sequence":"6"}`,
		accountNumber)

	return []byte(tmp)
}

func TestSignaturesHD(t *testing.T) {
	for account := uint32(0); account < 100; account += 30 {
		msg := getFakeTx(account)

		path := *hd.NewFundraiserParams(account, sdk.GetConfig().GetCoinType(), account/5)
		t.Logf("Checking signature at %v    ---   PLEASE REVIEW AND ACCEPT IN THE DEVICE\n", path)

		priv, err := NewPrivKeySecp256k1Unsafe(path)
		require.NoError(t, err)

		pub := priv.PubKey()
		sig, err := priv.Sign(msg)
		require.NoError(t, err)

		valid := pub.VerifySignature(msg, sig)
		require.True(t, valid, "Is your device using test mnemonic: %s ?", testdata.TestMnemonic)
	}
}

func TestRealDeviceSecp256k1(t *testing.T) {
	msg := getFakeTx(50)
	path := *hd.NewFundraiserParams(0, sdk.GetConfig().GetCoinType(), 0)
	priv, err := NewPrivKeySecp256k1Unsafe(path)
	require.NoError(t, err)

	pub := priv.PubKey()
	sig, err := priv.Sign(msg)
	require.NoError(t, err)

	valid := pub.VerifySignature(msg, sig)
	require.True(t, valid)

	// now, let's serialize the public key and make sure it still works
	bs := cdc.Amino.MustMarshalBinaryBare(priv.PubKey())
	pub2, err := legacy.PubKeyFromBytes(bs)
	require.Nil(t, err, "%+v", err)

	// make sure we get the same pubkey when we load from disk
	require.Equal(t, pub, pub2)

	// signing with the loaded key should match the original pubkey
	sig, err = priv.Sign(msg)
	require.NoError(t, err)
	valid = pub.VerifySignature(msg, sig)
	require.True(t, valid)

	// make sure pubkeys serialize properly as well
	bs = legacy.Cdc.MustMarshal(pub)
	bpub, err := legacy.PubKeyFromBytes(bs)
	require.NoError(t, err)
	require.Equal(t, pub, bpub)
}
