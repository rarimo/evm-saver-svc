package voting

import (
	"context"
	"fmt"
	"strconv"
	"time"

	events2 "github.com/rarimo/evm-saver-svc/internal/rarimo/events"
	oracletypes "github.com/rarimo/rarimo-core/x/oraclemanager/types"
	"gitlab.com/distributed_lab/running"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	gobind "github.com/rarimo/evm-bridge-contracts/gobind/contracts/interfaces/handlers"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/rarimo"
	rarimocore "github.com/rarimo/rarimo-core/x/rarimocore/types"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
	"github.com/rarimo/saver-grpc-lib/voter"
	"github.com/rarimo/saver-grpc-lib/voter/verifiers"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

const (
	IERC20DepositedTopic   = "0x043d52f9acdd847f0210803c386559db9e09d492143f2072fe30ea62ff0bb639"
	IERC721DepositedTopic  = "0x7f787dd0c844dac4f8bfc4044046cdab3be531f7eefa9b740c531e48a99725e1"
	IERC1155DepositedTopic = "0x103b790f2fa3a8676ff87c3620a55f0853d0e45128a8c7e9fadf29e17c51d07a"
	INativeDepositedTopic  = "0x9a47c8733424880a9e86a368eff95da5e7d36b68474a95eb097be2e43c116f27"

	EventNameTopic = 0
)

type ReceiptsProvider interface {
	GetTxReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error)
}

type IERC20Parser interface {
	ParseDepositedERC20(log types.Log) (*gobind.IERC20HandlerDepositedERC20, error)
}

type INativeParser interface {
	ParseDepositedNative(log types.Log) (*gobind.INativeHandlerDepositedNative, error)
}

type EvmTransferVerifier struct {
	log       *logan.Entry
	homeChain string

	receiptsProvider ReceiptsProvider
	parser20         IERC20Parser
	parserNative     INativeParser

	oracleQueryClient oracletypes.QueryClient
	tokenQueryClient  tokentypes.QueryClient
	msger             *rarimo.MessageMaker
}

func RunVoter(ctx context.Context, cfg config.Config) {
	v := voter.NewVoter(cfg.Ethereum().NetworkName, cfg.Log(), cfg.Broadcaster(), map[rarimocore.OpType]voter.Verifier{
		rarimocore.OpType_TRANSFER: verifiers.NewTransferVerifier(
			NewTransfersVerifier(cfg),
			cfg.Log().WithField("who", "evm-transfer-verifier"),
		),
	})

	// catchup tends to panic on startup and doesn't handle it by itself, so we wrap it into retry loop
	running.UntilSuccess(ctx, cfg.Log(), "voter-catchup", func(ctx context.Context) (bool, error) {
		voter.
			NewCatchupper(cfg.Cosmos(), v, cfg.Log()).
			Run(ctx)

		return true, nil
	}, 1*time.Second, 5*time.Second)

	// run blocking verification subscription
	voter.
		NewTransferSubscriber(v, cfg.Tendermint(), cfg.Cosmos(), cfg.Log(), cfg.Subscriber()).
		Run(ctx)
}

func NewTransfersVerifier(cfg config.Config) *EvmTransferVerifier {
	erc20Filterer, err := gobind.NewIERC20HandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init erc20 filterer"))
	}

	nativeFilterer, err := gobind.NewINativeHandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init native filterer"))
	}

	return &EvmTransferVerifier{
		log:               cfg.Log(),
		homeChain:         cfg.Ethereum().NetworkName,
		oracleQueryClient: oracletypes.NewQueryClient(cfg.Cosmos()),
		tokenQueryClient:  tokentypes.NewQueryClient(cfg.Cosmos()),
		receiptsProvider:  cfg.Ethereum().TxProvider,
		parser20:          erc20Filterer,
		parserNative:      nativeFilterer,
		msger:             rarimo.NewMessageMaker(cfg),
	}
}

func (e *EvmTransferVerifier) VerifyTransfer(ctx context.Context, txHash, eventId string, transfer *rarimocore.Transfer) error {
	if transfer.From.Chain != e.homeChain {
		return verifiers.ErrUnsupportedNetwork
	}

	txReceipt, err := e.receiptsProvider.GetTxReceipt(ctx, common.HexToHash(txHash)) // FIXME(hp): NEED A CONTEXT AS A PARAMETER
	if err != nil {
		return errors.Wrap(err, "failed to get transaction", logan.F{
			"tx_hash": txHash,
		})
	}

	logID, err := strconv.Atoi(eventId)
	if err != nil {
		return errors.Wrap(err, "failed to parse event id")
	}

	var eventLog *types.Log

	for _, log := range txReceipt.Logs {
		if log.Index != uint(logID) {
			continue
		}

		eventLog = log
		break
	}

	if eventLog == nil {
		return errors.From(errors.New("not found log in tx receipt"), logan.F{
			"tx":     txHash,
			"log_id": logID,
		})
	}

	switch eventLog.Topics[EventNameTopic].Hex() { // I wish abigen could generate generic code
	case IERC20DepositedTopic:
		event, err := e.parser20.ParseDepositedERC20(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}

		msg, err := e.msger.TransferMsg(ctx, &events2.IERC20Event{E: event})
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return e.checkTransferAtCore(ctx, msg, transfer)
	case INativeDepositedTopic: // hack for making native contract distinguishable
		event, err := e.parserNative.ParseDepositedNative(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}

		msg, err := e.msger.TransferMsg(ctx, &events2.INativeEvent{E: event})
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return e.checkTransferAtCore(ctx, msg, transfer)
	default:
		return fmt.Errorf("unsupported topic: %s", eventLog.Topics[0].Hex())
	}
}

func (e *EvmTransferVerifier) checkTransferAtCore(ctx context.Context,
	msgToQuery *oracletypes.MsgCreateTransferOp,
	transferToCheck *rarimocore.Transfer,
) error {
	transferResp, err := e.oracleQueryClient.Transfer(ctx, &oracletypes.QueryGetTransferRequest{Msg: *msgToQuery})
	if err != nil {
		return errors.Wrap(err, "error querying transfer from core")
	}

	if !proto.Equal(&transferResp.Transfer, transferToCheck) {
		return verifiers.ErrWrongOperationContent
	}

	return nil
}
