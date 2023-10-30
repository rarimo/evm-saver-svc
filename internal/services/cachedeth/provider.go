package cachedeth

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/vmihailenco/msgpack"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type ReceiptPayload struct {
	Receipt types.Receipt `json:"receipt"`
}

func (p *ReceiptPayload) UnmarshalBinary(data []byte) error {
	var pp ReceiptPayload
	if err := msgpack.Unmarshal(data, &pp); err != nil {
		return errors.Wrap(err, "failed to msgpack unmarshal")
	}

	*p = pp
	return nil
}

func (p *ReceiptPayload) MarshalBinary() (data []byte, err error) {
	return msgpack.Marshal(p)
}

type TxPayload struct {
	Tx     types.Transaction `json:"tx"`
	Sender string            `json:"sender"`
}

func (p *TxPayload) UnmarshalBinary(data []byte) error {
	var pp TxPayload
	if err := msgpack.Unmarshal(data, &pp); err != nil {
		return errors.Wrap(err, "failed to msgpack unmarshal")
	}

	*p = pp
	return nil
}

func (p *TxPayload) MarshalBinary() (data []byte, err error) {
	return msgpack.Marshal(p)
}

type Provider struct {
	log    *logan.Entry
	client *ethclient.Client
}

func NewProvider(log *logan.Entry, client *ethclient.Client) (*Provider, error) {
	return &Provider{
		log:    log,
		client: client,
	}, nil
}

func (p *Provider) GetTxReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	liveReceipt, err := p.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tx receipt by hash", logan.F{
			"hash": hash,
		})
	}

	return liveReceipt, nil
}

func (p *Provider) GetTx(ctx context.Context, hash common.Hash) (*types.Transaction, string, error) {
	tx, _, err := p.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get tx by hash", logan.F{
			"hash": hash,
		})
	}

	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get tx sender", logan.F{
			"tx_hash": hash,
		})
	}

	return tx, sender.String(), nil
}
