package evm

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/rarimo"
	events2 "github.com/rarimo/evm-saver-svc/internal/rarimo/events"
	"github.com/rarimo/saver-grpc-lib/metrics"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/distributed_lab/running"
)

func RunNativeListener(ctx context.Context, cfg config.Config) {
	const runnerName = "inative_listener"

	log := cfg.Log().WithField("who", runnerName)

	handler, err := gobind.NewINativeHandler(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init native handler"))
	}

	listener := nativeListener{
		listener: newListener(cfg),
		handler:  handler,
		msger:    rarimo.NewMessageMaker(cfg),
	}

	running.WithBackOff(ctx, log, runnerName,
		listener.subscription,
		5*time.Second, 5*time.Second, 5*time.Second)
}

type nativeListener struct {
	*listener
	handler *gobind.INativeHandler
	msger   *rarimo.MessageMaker
}

func (l *nativeListener) subscription(ctx context.Context) error {
	lastBlock, err := l.blockHandler.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get recent block")
	}

	lastBlock -= l.blockWindow

	if lastBlock < l.fromBlock {
		l.log.Infof("Skipping window: start %d > finish %d", l.fromBlock, lastBlock)
		return nil
	}

	if l.fromBlock+MaxBlocksPerRequest < lastBlock {
		l.log.Debugf("maxBlockPerRequest limit exceeded: setting last block to %d instead of %d", l.fromBlock+MaxBlocksPerRequest, lastBlock)
		lastBlock = l.fromBlock + MaxBlocksPerRequest
	}

	l.log.Infof("Starting subscription from %d to %d", l.fromBlock, lastBlock)
	defer l.log.Info("Subscription finished")

	const chanelBufSize = 10
	sink := make(chan *gobind.INativeHandlerDepositedNative, chanelBufSize)
	defer close(sink)

	iter, err := l.handler.FilterDepositedNative(&bind.FilterOpts{
		Start:   l.fromBlock,
		End:     &lastBlock,
		Context: ctx,
	})

	if err != nil {
		metrics.WebsocketMetric.Set(metrics.WebsocketDisconnected)
		return errors.Wrap(err, "failed to filter native deposit events")
	}

	defer func() {
		// https://ethereum.stackexchange.com/questions/8199/are-both-the-eth-newfilter-from-to-fields-inclusive
		// End in FilterLogs is inclusive
		l.fromBlock = lastBlock + 1
	}()

	metrics.WebsocketMetric.Set(metrics.WebsocketAvailable)

	for iter.Next() {
		e := iter.Event

		if e == nil {
			l.log.Error("got nil event")
			continue
		}

		l.log.WithFields(logan.F{
			"tx_hash":   e.Raw.TxHash,
			"tx_index":  e.Raw.TxIndex,
			"log_index": e.Raw.Index,
		}).Debug("got event")

		err := rarimo.MakeAndBroadcastMsg(ctx, l.msger, l.broadcaster, &events2.INativeEvent{E: e})
		if err != nil {
			return errors.Wrap(err, "failed to process event")
		}
	}
	return nil
}
