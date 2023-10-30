package evmmeta

import (
	"context"
	"math/big"

	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
)

var ErrMethodNotSupported = errors.New("method not supported")

// todo make it cached
type contractMetadata struct {
	metadata          gobind.Metadata
	contractInterface gobind.ContractInterface
}

func NewContractMetadata(addr common.Address, ci gobind.ContractInterface, caller bind.ContractCaller) (*contractMetadata, error) {
	switch ci {
	case gobind.ContractInterfaceERC20, gobind.ContractInterfaceERC721, gobind.ContractInterfaceERC1155:
		// bup
	default:
		return nil, errors.New("unknown contract interface")
	}

	return &contractMetadata{
		contractInterface: ci,
		metadata:          gobind.NewContractMetadata(addr, ci, caller),
	}, nil
}

func (p *contractMetadata) Name(ctx context.Context) (string, error) {
	result, err := p.metadata.Name(ctx)
	if errors.Cause(err) == gobind.ErrMethodNotSupported {
		return "", nil // not critical, so we can ignore
	}

	return result, err

}

func (p *contractMetadata) Symbol(ctx context.Context) (string, error) {
	result, err := p.metadata.Symbol(ctx)
	if errors.Cause(err) == gobind.ErrMethodNotSupported {
		return "", nil // not critical, so we can ignore
	}

	return result, err
}

func (p *contractMetadata) TokenURI(ctx context.Context, tokenID *big.Int) (string, error) {
	result, err := p.metadata.TokenURI(ctx, tokenID)
	if errors.Cause(err) == gobind.ErrMethodNotSupported {
		return "", ErrMethodNotSupported
	}

	return result, err

}

func (p *contractMetadata) ContractURI(ctx context.Context) (string, error) {
	result, err := p.metadata.ContractURI(ctx)
	if errors.Cause(err) == gobind.ErrMethodNotSupported {
		return "", nil
	}

	return result, err
}

func (p *contractMetadata) Decimals(ctx context.Context) (uint8, error) {
	result, err := p.metadata.Decimals(ctx)
	if errors.Cause(err) == gobind.ErrMethodNotSupported {
		return 0, ErrMethodNotSupported
	}

	return result, err
}

func (p *contractMetadata) CollectionTokenType() tokentypes.Type {
	switch p.contractInterface {
	case gobind.ContractInterfaceERC20:
		return tokentypes.Type_ERC20
	case gobind.ContractInterfaceERC721:
		return tokentypes.Type_ERC721
	case gobind.ContractInterfaceERC1155:
		return tokentypes.Type_ERC1155
	default:
		// normally should never happen
		panic("unknown contract interface")
	}
}
