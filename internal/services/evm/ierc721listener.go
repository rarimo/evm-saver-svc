package evm

import (
	"context"
	"time"

	"gitlab.com/rarimo/savers/saver-grpc-lib/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/ethtorarimo"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/distributed_lab/running"
	gobind "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc721"
)

func RunIERC721Listener(ctx context.Context, cfg config.Config) {
	const runnerName = "ierc721_listener"

	log := cfg.Log().WithField("who", runnerName)

	handler, err := gobind.NewIERC721Handler(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init erc721 handler"))
	}

	listener := ierc721Listener{
		listener: newListener(cfg),
		handler:  handler,
		msger:    ethtorarimo.CreateMessageMaker[*ethtorarimo.IERC721Event](cfg),
	}

	running.WithBackOff(ctx, log, runnerName,
		listener.subscription,
		5*time.Second, 5*time.Second, 5*time.Second)
}

type ierc721Listener struct {
	*listener
	handler *gobind.IERC721Handler
	msger   ethtorarimo.TxMsger[*ethtorarimo.IERC721Event]
}

func (l *ierc721Listener) subscription(ctx context.Context) error {
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
	sink := make(chan *gobind.IERC721HandlerDepositedERC721, chanelBufSize)
	defer close(sink)

	iter, err := l.handler.FilterDepositedERC721(&bind.FilterOpts{
		Start:   l.fromBlock,
		End:     &lastBlock,
		Context: ctx,
	})

	if err != nil {
		metrics.WebsocketMetric.Set(metrics.WebsocketDisconnected)
		return errors.Wrap(err, "failed to filter erc721 deposit events")
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

		err := ethtorarimo.MakeAndBroadcastMsg(ctx, l.msger, l.broadcaster, &ethtorarimo.IERC721Event{e})
		if err != nil {
			return errors.Wrap(err, "failed to process event")
		}
	}
	return nil
}
