package evmmeta

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
)

type MetadataClient interface {
	LoadMetadata(ctx context.Context, uri string, payload interface{}) error
	DownloadImage(ctx context.Context, imageURL string) ([]byte, error)
}

type Provider struct {
	client         *ethclient.Client
	metadataClient MetadataClient
}

func NewProvider(client *ethclient.Client, metadataClient MetadataClient) *Provider {
	return &Provider{client: client, metadataClient: metadataClient}
}

func (p *Provider) ContractMetadata(address common.Address, ci gobind.ContractInterface) (metadata.ContractMetadata, error) {
	return NewContractMetadata(address, ci, p.client)
}
