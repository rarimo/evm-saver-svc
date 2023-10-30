package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
)

var ZeroAddr = common.HexToAddress("0x0000000000000000000000000000000000000000")

type Event interface {
	Raw() types.Log
	Token() common.Address
	TokenId() *big.Int
	Amount() *big.Int
	Salt() [32]byte
	Bundle() []byte
	Network() string
	Receiver() string
	IsWrapped() bool
	TokenType() tokentypes.Type
	OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex
}
