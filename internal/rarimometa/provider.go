package rarimometa

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type Provider struct {
	log              *logan.Entry
	metadataProvider metadata.Provider
	tokenQueryClient tokentypes.QueryClient
}

func NewProvider(
	log *logan.Entry,
	metadataProvider metadata.Provider,
	tokenQueryClient tokentypes.QueryClient,
) *Provider {
	return &Provider{
		log:              log,
		metadataProvider: metadataProvider,
		tokenQueryClient: tokenQueryClient,
	}
}

func (p *Provider) EthItemMeta(ctx context.Context,
	ci gobind.ContractInterface,
	contractAddr common.Address,
	tokenID *big.Int,
) (*tokentypes.ItemMetadata, error) {
	meta, err := p.metadataProvider.TokenMetadata(ctx, ci, contractAddr, tokenID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token metadata")
	}

	imgHash, err := p.generateImageHash(ctx, meta.TokenMeta.ImageURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate image hash")
	}

	return &tokentypes.ItemMetadata{
		ImageUri:  meta.TokenMeta.ImageURI,
		ImageHash: imgHash,
		Uri:       meta.URI,
	}, nil
}

func (p *Provider) generateImageHash(ctx context.Context, imageURL string) (string, error) {
	rawImg, err := p.metadataProvider.DownloadImage(ctx, imageURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to download image")
	}

	hash := sha256.Sum256(rawImg)
	hash64 := base64.StdEncoding.EncodeToString(hash[:])

	return hash64, nil
}
