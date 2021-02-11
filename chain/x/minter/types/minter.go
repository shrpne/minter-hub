package types

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/mhub/chain/coins"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"regexp"
)

const (
	// PeggyDenomPrefix indicates the prefix for all assests minted by this module
	PeggyDenomPrefix = ModuleName

	// PeggyDenomSeperator is the seperator for peggy denoms
	PeggyDenomSeperator = "/"

	// MinterMultisigAddressLen is the length of multisig address strings
	MinterMultisigAddressLen = 42

	// PeggyDenomLen is the length of the denoms generated by the peggy module
	PeggyDenomLen = len(PeggyDenomPrefix) + len(PeggyDenomSeperator) + MinterMultisigAddressLen
)

// MinterAddrLessThan migrates the Minter address less than function
func MinterAddrLessThan(e, o string) bool {
	return bytes.Compare([]byte(e)[:], []byte(o)[:]) == -1
}

// ValidateMinterAddress validates the minter address strings
func ValidateMinterAddress(a string) error {
	if a == "" {
		return fmt.Errorf("empty")
	}
	if !regexp.MustCompile("^Mx[0-9a-fA-F]{40}$").MatchString(a) {
		return fmt.Errorf("address(%s) doesn't pass regex", a)
	}
	if len(a) != MinterMultisigAddressLen {
		return fmt.Errorf("address(%s) of the wrong length exp(%d) actual(%d)", a, len(a), MinterMultisigAddressLen)
	}
	return nil
}

/////////////////////////
//     ERC20Token      //
/////////////////////////

// NewMinterCoin returns a new instance of an ERC20
func NewMinterCoin(amount sdk.Int, coinId uint64) *MinterCoin {
	return &MinterCoin{Amount: amount, CoinId: coinId}
}

// PeggyCoin returns the peggy representation of the ERC20
func (e *MinterCoin) PeggyCoin() sdk.Coin {
	coinsList := coins.GetCoins()
	for _, coin := range coinsList {
		if coin.MinterID == e.CoinId {
			return sdk.NewCoin(coin.Denom, e.Amount)
		}
	}
	return sdk.NewCoin(fmt.Sprintf("%s/%d", PeggyDenomPrefix, e.CoinId), e.Amount)
}

// ValidateBasic permforms stateless validation
func (e *MinterCoin) ValidateBasic() error {
	// TODO: Validate all the things
	return nil
}

// Add adds one ERC20 to another
// TODO: make this return errors instead
func (e *MinterCoin) Add(o *MinterCoin) *MinterCoin {
	if e.CoinId != o.CoinId {
		panic("invalid coins")
	}

	sum := e.Amount.Add(o.Amount)
	if !sum.IsUint64() {
		panic("invalid amount")
	}
	return NewMinterCoin(sum, e.CoinId)
}

// MinterCoinFromPeggyCoin returns the ERC20 representation of a given peggy coin
func MinterCoinFromPeggyCoin(v sdk.Coin) (*MinterCoin, error) {
	coinId, err := ValidatePeggyCoin(v)
	if err != nil {
		return nil, fmt.Errorf("%s isn't a valid peggy coin: %s", v.String(), err)
	}
	return &MinterCoin{CoinId: coinId, Amount: v.Amount}, nil
}

// ValidatePeggyCoin returns true if a coin is a peggy representation of an Minter Coin
func ValidatePeggyCoin(v sdk.Coin) (uint64, error) {
	coinsList := coins.GetCoins()
	for _, coin := range coinsList {
		if v.Denom == coin.Denom {
			return coin.MinterID, nil
		}
	}

	return 0, fmt.Errorf("denom(%s) not valid", v.Denom)
}
