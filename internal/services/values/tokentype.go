package values

type TokenType int32

const (
	TokenTypeERC20 TokenType = iota
	TokenTypeERC721
	TokenTypeERC1155
	TokenTypeNative
)
