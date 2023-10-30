package ethtorarimo

import (
	"context"
	"net/http"
	"time"

	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/evmmeta"
	"github.com/rarimo/evm-saver-svc/internal/rarimo"
	"github.com/rarimo/evm-saver-svc/internal/rarimometa"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	oracletypes "gitlab.com/rarimo/rarimo-core/x/oraclemanager/types"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarimo/savers/saver-grpc-lib/broadcaster"
)

type TxMsger[T rarimo.Event] interface {
	TransferMsg(ctx context.Context, event T) (*oracletypes.MsgCreateTransferOp, error)
}

func MakeAndBroadcastMsg[T rarimo.Event](ctx context.Context, msger TxMsger[T], bc broadcaster.Broadcaster, event T) error {
	msg, err := msger.TransferMsg(ctx, event)
	if err != nil {
		return errors.Wrap(err, "failed to craft transfer msg", logan.F{
			"tx_hash": event.Raw().TxHash.String(),
		})
	}

	return bc.BroadcastTx(ctx, msg)
}

func CreateMessageMaker[T rarimo.Event](cfg config.Config) *rarimo.MessageMaker[T] {
	return rarimo.NewMessageMaker[T](
		cfg.Log().WithField("who", "message-maker"),
		cfg.Broadcaster().Sender(),
		cfg.Ethereum().NetworkName,
		tokentypes.NewQueryClient(cfg.Cosmos()),
		rarimometa.NewProvider(
			cfg.Log().WithField("who", "rarimo-meta-provider"),
			evmmeta.NewProvider(
				cfg.Ethereum().RPCClient,
				metadata.NewClient(http.DefaultClient, cfg.IPFS(), 1*time.Minute),
			),
			tokentypes.NewQueryClient(cfg.Cosmos()),
		),
		cfg.Ethereum().TxProvider,
	)
}
