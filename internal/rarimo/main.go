package rarimo

import (
	"context"

	"github.com/rarimo/evm-saver-svc/internal/rarimo/events"
	"github.com/rarimo/saver-grpc-lib/broadcaster"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

func MakeAndBroadcastMsg(ctx context.Context, msger *MessageMaker, bc broadcaster.Broadcaster, event events.Event) error {
	msg, err := msger.TransferMsg(ctx, event)
	if err != nil {
		return errors.Wrap(err, "failed to craft transfer msg", logan.F{
			"tx_hash": event.Raw().TxHash.String(),
		})
	}

	return bc.BroadcastTx(ctx, msg)
}
