package evmmeta

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
)

func (p *Provider) TokenMetadata(ctx context.Context, ci gobind.ContractInterface, addr common.Address, tokenID *big.Int) (*metadata.Payload, error) {
	contractMetadata, err := p.ContractMetadata(addr, ci)
	if err != nil {
		return nil, err
	}

	cName, err := contractMetadata.Name(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load contract name")
	}

	cSymbol, err := contractMetadata.Symbol(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load contract symbol")
	}

	var uri string

	switch ci {
	case gobind.ContractInterfaceERC20:
		// FIXME should it be stored in contract uri?
		if uri, err = contractMetadata.ContractURI(ctx); err != nil {
			return nil, errors.Wrap(err, "failed to get contract uri")
		}
	case gobind.ContractInterfaceERC721, gobind.ContractInterfaceERC1155:
		if uri, err = contractMetadata.TokenURI(ctx, tokenID); err != nil {
			return nil, errors.Wrap(err, "failed to get token uri")
		}
	default:
		return nil, errors.New("unsupported contract interface")
	}

	payload := metadata.Payload{
		ContractMeta: metadata.ContractPayload{
			Name:   cName,
			Symbol: cSymbol,
		},
	}

	if err := p.metadataClient.LoadMetadata(ctx, uri, &payload); err != nil {
		return nil, errors.Wrap(err, "failed to load metadata")
	}

	return &payload, nil
}

func (p *Provider) DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	return p.metadataClient.DownloadImage(ctx, imageURL)
}
