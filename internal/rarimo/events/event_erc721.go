package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type IERC721Event struct {
	E *gobind.IERC721HandlerDepositedERC721
}

func (e *IERC721Event) Token() common.Address {
	return e.E.Token
}

func (e *IERC721Event) TokenId() *big.Int {
	return e.E.TokenId
}

func (e *IERC721Event) Amount() *big.Int {
	return big.NewInt(1)
}

func (e *IERC721Event) Salt() [32]byte {
	return e.E.Salt
}

func (e *IERC721Event) Bundle() []byte {
	return e.E.Bundle
}

func (e *IERC721Event) Network() string {
	return e.E.Network
}

func (e *IERC721Event) Receiver() string {
	return e.E.Receiver
}

func (e *IERC721Event) IsWrapped() bool {
	return e.E.IsWrapped
}

func (e *IERC721Event) Raw() types.Log {
	return e.E.Raw
}

func (e *IERC721Event) TokenType() tokentypes.Type {
	return tokentypes.Type_ERC721
}

func (e *IERC721Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
		TokenID: e.E.TokenId.String(),
	}
}
