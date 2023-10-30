package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type IERC20Event struct {
	E *gobind.IERC20HandlerDepositedERC20
}

func (e *IERC20Event) Token() common.Address {
	return e.E.Token
}

func (e *IERC20Event) TokenId() *big.Int {
	return big.NewInt(0)
}

func (e *IERC20Event) Amount() *big.Int {
	return e.E.Amount
}

func (e *IERC20Event) Salt() [32]byte {
	return e.E.Salt
}

func (e *IERC20Event) Bundle() []byte {
	return e.E.Bundle
}

func (e *IERC20Event) Network() string {
	return e.E.Network
}

func (e *IERC20Event) Receiver() string {
	return e.E.Receiver
}

func (e *IERC20Event) IsWrapped() bool {
	return e.E.IsWrapped
}

func (e *IERC20Event) Raw() types.Log {
	return e.E.Raw
}

func (e *IERC20Event) TokenType() tokentypes.Type {
	return tokentypes.Type_ERC20
}

func (e *IERC20Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
	}
}
