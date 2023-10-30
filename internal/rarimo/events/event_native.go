package events

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type INativeEvent struct {
	E *gobind.INativeHandlerDepositedNative
}

func (e *INativeEvent) Token() common.Address {
	return ZeroAddr
}

func (e *INativeEvent) TokenId() *big.Int {
	return big.NewInt(0)
}

func (e *INativeEvent) Amount() *big.Int {
	return e.E.Amount
}

func (e *INativeEvent) Salt() [32]byte {
	return e.E.Salt
}

func (e *INativeEvent) Bundle() []byte {
	return e.E.Bundle
}

func (e *INativeEvent) Network() string {
	return e.E.Network
}

func (e *INativeEvent) Receiver() string {
	return e.E.Receiver
}

func (e *INativeEvent) IsWrapped() bool {
	return false
}

func (e *INativeEvent) Raw() types.Log {
	return e.E.Raw
}

func (e *INativeEvent) TokenType() tokentypes.Type {
	return tokentypes.Type_NATIVE
}

func (e *INativeEvent) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: "",
	}
}
