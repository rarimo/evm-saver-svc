package ethtorarimo

import (
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
)

func (e *INativeEvent) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: "",
	}
}

func (e *INativeEvent) TokenInterface() gobind.ContractInterface {
	return gobind.ContractInterfaceERC20
}

func (e *IERC20Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
	}
}

func (e *IERC721Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
		TokenID: e.E.TokenId.String(),
	}
}

func (e *IERC1155Event) OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex {
	return &tokentypes.OnChainItemIndex{
		Chain:   onNetwork,
		Address: e.E.Token.String(),
		TokenID: e.E.TokenId.String(),
	}
}
