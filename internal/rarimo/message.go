package rarimo

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/rarimo/events"
	oracletypes "github.com/rarimo/rarimo-core/x/oraclemanager/types"
	tokentypes "github.com/rarimo/rarimo-core/x/tokenmanager/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EthTxProvider interface {
	GetTx(ctx context.Context, hash common.Hash) (*types.Transaction, string, error)
}

type MessageMaker struct {
	log              *logan.Entry
	txCreatorAddr    string
	homeChain        string
	tokenQueryClient tokentypes.QueryClient
	txProvider       EthTxProvider
}

func NewMessageMaker(
	cfg config.Config,
) *MessageMaker {
	return &MessageMaker{
		log:              cfg.Log(),
		txCreatorAddr:    cfg.Broadcaster().Sender(),
		homeChain:        cfg.Ethereum().NetworkName,
		tokenQueryClient: tokentypes.NewQueryClient(cfg.Cosmos()),
		txProvider:       cfg.Ethereum().TxProvider,
	}
}

func (m *MessageMaker) TransferMsg(ctx context.Context, event events.Event) (*oracletypes.MsgCreateTransferOp, error) {
	_, sender, err := m.txProvider.GetTx(ctx, event.Raw().TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eth tx")
	}

	srcItemIndex, dstItemIndex, err := m.itemOnChainIndices(ctx, event)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create on-chain item indexes", logan.F{
			"src_chain":   m.homeChain,
			"src_tx_hash": event.Raw().TxHash,
			"dst_chain":   event.Network(),
		})
	}

	msg := oracletypes.MsgCreateTransferOp{
		Tx:       event.Raw().TxHash.String(),
		Creator:  m.txCreatorAddr,
		EventId:  fmt.Sprintf("%d", event.Raw().Index),
		Sender:   sender,
		Receiver: event.Receiver(),
		Amount:   fmt.Sprint(event.Amount()),
		From:     *srcItemIndex,
		To:       *dstItemIndex,
	}

	if len(event.Bundle()) > 0 && notZero32(event.Salt()) {
		salt := event.Salt()
		msg.BundleSalt = hexutil.Encode(salt[:])
		msg.BundleData = hexutil.Encode(event.Bundle())
	}

	return &msg, nil
}

func (m *MessageMaker) itemOnChainIndices(ctx context.Context, e events.Event) (
	onChainItemSrc *tokentypes.OnChainItemIndex,
	onChainItemDst *tokentypes.OnChainItemIndex,
	err error,
) {
	onChainItemSrc = e.OnChainItemIndex(m.homeChain)

	onChainItemDst, err = m.ensureDstItem(ctx, onChainItemSrc, e.Network())
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to ensure destination item", logan.F{
			"src_chain":         onChainItemSrc.Chain,
			"src_token_id":      onChainItemSrc.TokenID,
			"src_contract_addr": onChainItemSrc.Address,
			"dst_chain":         e.Network(),
		})
	}

	return
}

func (m *MessageMaker) ensureDstItem(ctx context.Context, srcOnChainItem *tokentypes.OnChainItemIndex, dstChain string) (*tokentypes.OnChainItemIndex, error) {
	// pushing luck trying to fetch on chain item index for destination chain
	dstItemResp, err := m.tokenQueryClient.OnChainItemByOther(ctx, &tokentypes.QueryGetOnChainItemByOtherRequest{
		Chain:       m.homeChain,
		Address:     srcOnChainItem.Address,
		TokenID:     srcOnChainItem.TokenID,
		TargetChain: dstChain,
	})
	if err != nil {
		if res, ok := status.FromError(err); ok && res.Code() == codes.NotFound {
			return nil, errors.Wrap(err, "destination onc chain item not found")
		}

		return nil, errors.Wrap(err, "failed to fetch destination on chain item")
	}

	return dstItemResp.Item.Index, nil
}

func notZero32(bb [32]byte) bool {
	for _, b := range bb {
		if b != 0 {
			return true
		}
	}
	return false
}
