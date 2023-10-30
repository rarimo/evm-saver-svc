package ethtorarimo

import (
	"math/big"

	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rarimo/evm-saver-svc/internal/rarimo"
	"github.com/rarimo/evm-saver-svc/internal/services/values"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	gobind1155 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc1155"
	gobind20 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc20"
	gobind721 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc721"
	gobindnative "gitlab.com/rarimo/contracts/evm-bridge/gobind/inative"
)

var ZeroAddr = common.HexToAddress("0x0000000000000000000000000000000000000000")

type IERC20Event struct {
	E *gobind20.IERC20HandlerDepositedERC20
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

func (e *IERC20Event) TokenParams() rarimo.TokenParams {
	return rarimo.TokenParams{
		ContractInterface: gobind.ContractInterfaceERC20,
		TokenType:         values.TokenTypeERC20,
		CoreTokenType:     tokentypes.Type_ERC20,
	}
}

type INativeEvent struct {
	E *gobindnative.INativeHandlerDepositedNative
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

func (e *INativeEvent) TokenParams() rarimo.TokenParams {
	return rarimo.TokenParams{
		ContractInterface: [4]byte{0, 0, 0, 0}, // FIXME(hp): does it support ERC20 interface?
		TokenType:         values.TokenTypeNative,
		CoreTokenType:     tokentypes.Type_NATIVE,
	}
}

type IERC721Event struct {
	E *gobind721.IERC721HandlerDepositedERC721
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

func (e *IERC721Event) TokenParams() rarimo.TokenParams {
	return rarimo.TokenParams{
		ContractInterface: gobind.ContractInterfaceERC721,
		TokenType:         values.TokenTypeERC721,
		CoreTokenType:     tokentypes.Type_ERC721,
	}
}

type IERC1155Event struct {
	E *gobind1155.IERC1155HandlerDepositedERC1155
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

func (e *IERC1155Event) TokenParams() rarimo.TokenParams {
	return rarimo.TokenParams{
		ContractInterface: gobind.ContractInterfaceERC1155,
		TokenType:         values.TokenTypeERC1155,
		CoreTokenType:     tokentypes.Type_ERC1155,
	}
}
