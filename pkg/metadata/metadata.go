package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/vmihailenco/msgpack"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cast"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
)

type ContractPayload struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type TokenPayload struct {
	Name        string `json:"name,omitempty"` // on this level we care only about name, image and description fields
	Description string `json:"description,omitempty"`
	ImageURI    string `json:"image_uri,omitempty"`
}

type Payload struct {
	URI          string          `json:"uri"`
	RawMetadata  json.RawMessage `json:"raw_metadata"`
	TokenMeta    TokenPayload    `json:"token_meta"`
	ContractMeta ContractPayload `json:"contract_meta"`
}

func (p *Payload) UnmarshalBinary(data []byte) error {
	var pp Payload
	if err := msgpack.Unmarshal(data, &pp); err != nil {
		return errors.Wrap(err, "failed to msgpack unmarshal")
	}

	*p = pp
	return nil
}

func (p *Payload) MarshalBinary() (data []byte, err error) {
	return msgpack.Marshal(p)
}

var (
	imageAliases = []string{
		"image", "image_url", "image_uri", "image_link", "image_link_url", "image_link_uri", "image_url_cdn",
	}
	ErrNoImg     = fmt.Errorf("image not found in metadata by any of keys:[%+v]", imageAliases)
	ErrEmptyMeta = errors.New("metadata is empty")
)

func (p *Payload) ShouldCache() bool {
	return p.TokenMeta.ImageURI != "" // FIXME(hp): idk about other fields yet
}

func (p *Payload) CacheTTL() time.Duration {
	return 24 * time.Hour
}

func (p *Payload) PopulateURI(uri string) {
	p.URI = uri
}

func (p *Payload) UnmarshalFrom(raw json.RawMessage) error {
	if len(raw) == 0 {
		p.RawMetadata = json.RawMessage(`{}`)
		return ErrEmptyMeta
	}

	var payload map[string]interface{}
	err := json.Unmarshal(raw, &payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal", logan.F{
			"raw": string(raw),
		})
	}

	for _, alias := range imageAliases {
		if rawImage, ok := payload[alias]; ok {
			p.TokenMeta.ImageURI, err = cast.ToStringE(rawImage)
			if err != nil {
				return errors.Wrap(err, "failed to cast image", logan.F{
					"raw_image": rawImage,
				})
			}
			break
		}
	}

	p.RawMetadata = raw

	if p.TokenMeta.ImageURI == "" {
		// in order to allow further processing of the metadata without the image
		return ErrNoImg
	}

	return nil
}

type ImageLoader interface {
	DownloadImage(ctx context.Context, imageURL string) ([]byte, error)
}

type Provider interface {
	ImageLoader
	TokenMetadata(ctx context.Context, ci gobind.ContractInterface, addr common.Address, tokenID *big.Int) (*Payload, error)
}

type ContractMetadata interface {
	// Name - returns name of the contract. Empty string should be returned for cases covered by ErrMethodNotSupported
	Name(ctx context.Context) (string, error)
	// Symbol - returns symbol of the contract. Empty string should be returned for cases covered by ErrMethodNotSupported
	Symbol(ctx context.Context) (string, error)
	// TokenURI - performs call to the contract to get URI that holds metadata for the tokenID
	// For cases covered by ErrMethodNotSupported - this error MUST be returned
	// Any template that allow to pass tokenID MUST be transformed into {id} template and returned
	TokenURI(ctx context.Context, tokenID *big.Int) (string, error)
	// ContractURI - returns URI that holds contracts metadata (collection image, description, etc)
	ContractURI(ctx context.Context) (string, error)
	// Decimals - returns decimals of the contract. 0 should be returned for cases covered by ErrMethodNotSupported
	Decimals(ctx context.Context) (uint8, error)
	// CollectionTokenType - returns token type of all tokens within the collection
	CollectionTokenType() tokentypes.Type
}
