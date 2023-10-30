package voting

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/distributed_lab/running"
	oracletypes "gitlab.com/rarimo/rarimo-core/x/oraclemanager/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/ethtorarimo"
	"github.com/rarimo/evm-saver-svc/internal/rarimo"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	bind1155 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc1155"
	bind20 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc20"
	bind721 "gitlab.com/rarimo/contracts/evm-bridge/gobind/ierc721"
	bindNative "gitlab.com/rarimo/contracts/evm-bridge/gobind/inative"
	rarimocore "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarimo/savers/saver-grpc-lib/voter"
	"gitlab.com/rarimo/savers/saver-grpc-lib/voter/verifiers"
)

const (
	IERC20DepositedTopic   = "0x043d52f9acdd847f0210803c386559db9e09d492143f2072fe30ea62ff0bb639"
	IERC721DepositedTopic  = "0x7f787dd0c844dac4f8bfc4044046cdab3be531f7eefa9b740c531e48a99725e1"
	IERC1155DepositedTopic = "0x103b790f2fa3a8676ff87c3620a55f0853d0e45128a8c7e9fadf29e17c51d07a"
	INativeDepositedTopic  = "0x9a47c8733424880a9e86a368eff95da5e7d36b68474a95eb097be2e43c116f27"

	EventNameTopic = 0
)

type TxMsger[T rarimo.Event] interface {
	TransferMsg(ctx context.Context, event T) (*oracletypes.MsgCreateTransferOp, error)
}

type ReceiptsProvider interface {
	GetTxReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error)
}

type ContractMetadataProvider interface {
	ContractMetadata(address common.Address, ci gobind.ContractInterface) (metadata.ContractMetadata, error)
}

type IERC20Parser interface {
	ParseDepositedERC20(log types.Log) (*bind20.IERC20HandlerDepositedERC20, error)
}

type IERC721Parser interface {
	ParseDepositedERC721(log types.Log) (*bind721.IERC721HandlerDepositedERC721, error)
}

type IERC1155Parser interface {
	ParseDepositedERC1155(log types.Log) (*bind1155.IERC1155HandlerDepositedERC1155, error)
}

type INativeParser interface {
	ParseDepositedNative(log types.Log) (*bindNative.INativeHandlerDepositedNative, error)
}

type evmTransferVerifier struct {
	cfg config.Config
	log *logan.Entry

	receiptsProvider  ReceiptsProvider
	homeChain         string
	oracleQueryClient oracletypes.QueryClient
	tokenQueryClient  tokentypes.QueryClient

	parser20     IERC20Parser
	parser721    IERC721Parser
	parser1155   IERC1155Parser
	parserNative INativeParser
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

func NewTransfersVerifier(cfg config.Config) *evmTransferVerifier {
	erc20Filterer, err := bind20.NewIERC20HandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init erc20 filterer"))
	}

	erc721Filterer, err := bind721.NewIERC721HandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init erc721 filterer"))
	}

	erc1155Filterer, err := bind1155.NewIERC1155HandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init erc1155 filterer"))
	}

	nativeFilterer, err := bindNative.NewINativeHandlerFilterer(cfg.Ethereum().ContractAddr, cfg.Ethereum().RPCClient)
	if err != nil {
		panic(errors.Wrap(err, "failed to init native filterer"))
	}

	return &evmTransferVerifier{
		cfg:               cfg,
		log:               cfg.Log(),
		homeChain:         cfg.Ethereum().NetworkName,
		oracleQueryClient: oracletypes.NewQueryClient(cfg.Cosmos()),
		tokenQueryClient:  tokentypes.NewQueryClient(cfg.Cosmos()),
		receiptsProvider:  cfg.Ethereum().TxProvider,
		parser20:          erc20Filterer,
		parser721:         erc721Filterer,
		parser1155:        erc1155Filterer,
		parserNative:      nativeFilterer,
	}
}

func (tv *evmTransferVerifier) VerifyTransfer(ctx context.Context, txHash, eventId string, transfer *rarimocore.Transfer) error {
	if transfer.From.Chain != tv.homeChain {
		return verifiers.ErrUnsupportedNetwork
	}

	txReceipt, err := tv.receiptsProvider.GetTxReceipt(ctx, common.HexToHash(txHash)) // FIXME(hp): NEED A CONTEXT AS A PARAMETER
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
		event, err := tv.parser20.ParseDepositedERC20(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}

		revent := ethtorarimo.IERC20Event{event}
		msg, err := ethtorarimo.
			CreateMessageMaker[*ethtorarimo.IERC20Event](tv.cfg).
			TransferMsg(ctx, &revent)
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return tv.checkTransferAtCore(ctx,
			&revent,
			msg,
			transfer)
	case IERC721DepositedTopic:
		event, err := tv.parser721.ParseDepositedERC721(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}
		revent := ethtorarimo.IERC721Event{event}

		msg, err := ethtorarimo.
			CreateMessageMaker[*ethtorarimo.IERC721Event](tv.cfg).
			TransferMsg(ctx, &revent)
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return tv.checkTransferAtCore(ctx, &revent, msg, transfer)
	case IERC1155DepositedTopic:
		event, err := tv.parser1155.ParseDepositedERC1155(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}

		revent := ethtorarimo.IERC1155Event{event}
		msg, err := ethtorarimo.
			CreateMessageMaker[*ethtorarimo.IERC1155Event](tv.cfg).
			TransferMsg(ctx, &revent)
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return tv.checkTransferAtCore(ctx, &revent, msg, transfer)
	case INativeDepositedTopic: // hack for making native contract distinguishable
		event, err := tv.parserNative.ParseDepositedNative(*eventLog)
		if err != nil {
			return errors.Wrap(verifiers.ErrWrongOperationContent, "failed to parse erc20 log")
		}

		revent := ethtorarimo.INativeEvent{event}
		msg, err := ethtorarimo.
			CreateMessageMaker[*ethtorarimo.INativeEvent](tv.cfg).
			TransferMsg(ctx, &revent)
		if err != nil {
			return errors.Wrap(err, "failed to make transfer msg")
		}

		return tv.checkTransferAtCore(ctx, &revent, msg, transfer)
	default:
		return fmt.Errorf("unsupported topic: %s", eventLog.Topics[0].Hex())
	}
}

func (tv *evmTransferVerifier) checkTransferAtCore(ctx context.Context,
	event rarimo.Event,
	msgToQuery *oracletypes.MsgCreateTransferOp,
	transferToCheck *rarimocore.Transfer,
) error {
	if rarimo.IsNFT(event.TokenParams().CoreTokenType) {
		msgToQuery.To.TokenID = transferToCheck.To.TokenID
	}

	// seed checking is supported only for ERC721
	if event.TokenParams().CoreTokenType == tokentypes.Type_ERC721 {
		existingSeed := transferToCheck.Meta.Seed // FIXME this should be figured out when creating the message

		solanaNetwork, err := tv.tokenQueryClient.NetworkParams(ctx, &tokentypes.QueryNetworkParamsRequest{Name: rarimo.SolanaChainName})
		if err != nil {
			return errors.Wrap(err, "failed to get network params", logan.F{
				"network": rarimo.SolanaChainName,
			})
		}

		if ok := verifiers.MustVerifyTokenSeed(solanaNetwork.Params.Contract, existingSeed); !ok {
			tv.log.Error("failed to verify seed")
			return verifiers.ErrWrongOperationContent
		}

		tv.log.WithFields(logan.F{
			"solana_contract": solanaNetwork.Params.Contract,
			"seed":            msgToQuery.Meta.Seed,
		}).Debug("pda ok")

		pda := verifiers.MustGetPDA(solanaNetwork.Params.Contract, existingSeed)
		if pda != transferToCheck.To.TokenID {
			tv.log.WithFields(logan.F{
				"core_pda": pda,
				"token_id": transferToCheck.To.TokenID,
			}).Error("PDA not equal to target token ID") // errors are not logged properly so logging here (TODO fix in the library)

			return verifiers.ErrWrongOperationContent
		}

		tv.log.WithFields(logan.F{
			"core_pda": pda,
			"token_id": transferToCheck.To.TokenID,
		}).Debug("pda ok")

		if _, err := tv.tokenQueryClient.Seed(ctx, &tokentypes.QueryGetSeedRequest{Seed: existingSeed}); err != nil {
			if status.Code(err) != codes.NotFound {
				tv.log.WithError(err).Error("failed to check seed existence", logan.F{
					"checked_seed": existingSeed,
				})

				return verifiers.ErrWrongOperationContent
			}

			tv.log.WithFields(logan.F{
				"checked_seed": existingSeed,
			}).Debug("seed does not exist yet so can be used, proceeding")
		}

		msgToQuery.Meta.Seed = transferToCheck.Meta.Seed
	}

	transferResp, err := tv.oracleQueryClient.Transfer(ctx, &oracletypes.QueryGetTransferRequest{Msg: *msgToQuery})
	if err != nil {
		return errors.Wrap(err, "error querying transfer from core")
	}

	if !proto.Equal(&transferResp.Transfer, transferToCheck) {
		return verifiers.ErrWrongOperationContent
	}

	return nil
}
