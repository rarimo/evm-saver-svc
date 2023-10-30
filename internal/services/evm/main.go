package evm

import (
	"context"

	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/saver-grpc-lib/broadcaster"
	"gitlab.com/distributed_lab/logan/v3"
)

const MaxBlocksPerRequest = 100

type blockHandler interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

type listener struct {
	log          *logan.Entry
	blockHandler blockHandler
	broadcaster  broadcaster.Broadcaster
	fromBlock    uint64
	blockWindow  uint64
}

func newListener(cfg config.Config) *listener {
	return &listener{
		log:          cfg.Log(),
		blockHandler: cfg.Ethereum().RPCClient,
		broadcaster:  cfg.Broadcaster(),
		fromBlock:    cfg.Ethereum().StartFromBlock,
		blockWindow:  cfg.Ethereum().BlockWindow,
	}
}
