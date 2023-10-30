package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type IERC1155Event struct {
	E *gobind.IERC1155HandlerDepositedERC1155
}

func (e *IERC1155Event) Token() common.Address {
	return e.E.Token
}

func (e *IERC1155Event) TokenId() *big.Int {
	return e.E.TokenId
}

func (e *IERC1155Event) Amount() *big.Int {
	return e.E.Amount
}

func (e *IERC1155Event) Salt() [32]byte {
	return e.E.Salt
}

func (e *IERC1155Event) Bundle() []byte {
	return e.E.Bundle
}

func (e *IERC1155Event) Network() string {
	return e.E.Network
}

func (e *IERC1155Event) Receiver() string {
	return e.E.Receiver
}

func (e *IERC1155Event) IsWrapped() bool {
	return e.E.IsWrapped
}

func (e *IERC1155Event) Raw() types.Log {
	return e.E.Raw
}

func (e *IERC1155Event) TokenType() tokentypes.Type {
	return tokentypes.Type_ERC1155
}

func (e *IERC1155Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
		TokenID: e.E.TokenId.String(),
	}
}
