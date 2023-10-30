package rarimo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rarimo/evm-saver-svc/internal/services/values"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
	oracletypes "gitlab.com/rarimo/rarimo-core/x/oraclemanager/types"
	tokentypes "gitlab.com/rarimo/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarimo/savers/saver-grpc-lib/voter/verifiers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	SolanaChainName = "Solana"
)

type EthTxProvider interface {
	GetTx(ctx context.Context, hash common.Hash) (*types.Transaction, string, error)
}

type TokenParams struct {
	ContractInterface gobind.ContractInterface
	TokenType         values.TokenType
	CoreTokenType     tokentypes.Type
}

type Event interface {
	Raw() types.Log
	Token() common.Address
	TokenId() *big.Int
	Amount() *big.Int
	Salt() [32]byte
	Bundle() []byte
	Network() string
	Receiver() string
	IsWrapped() bool
	TokenParams() TokenParams

	OnChainItemIndex(onNetwork string) *tokentypes.OnChainItemIndex
}

type metadataProvider interface {
	EthItemMeta(ctx context.Context, ci gobind.ContractInterface, contractAddr common.Address, tokenID *big.Int) (*tokentypes.ItemMetadata, error)
}

type MessageMaker[T Event] struct {
	log              *logan.Entry
	txCreatorAddr    string
	homeChain        string
	tokenQueryClient tokentypes.QueryClient
	metadataProvider metadataProvider
	txProvider       EthTxProvider
}

func NewMessageMaker[T Event](
	log *logan.Entry,
	txCreatorAddr string,
	homeChain string,
	tokenQueryClient tokentypes.QueryClient,
	metadataProvider metadataProvider,
	txProvider EthTxProvider,
) *MessageMaker[T] {
	return &MessageMaker[T]{
		log:              log,
		txCreatorAddr:    txCreatorAddr,
		homeChain:        homeChain,
		tokenQueryClient: tokenQueryClient,
		metadataProvider: metadataProvider,
		txProvider:       txProvider,
	}
}

func (m *MessageMaker[T]) TransferMsg(ctx context.Context, event T) (*oracletypes.MsgCreateTransferOp, error) {
	_, sender, err := m.txProvider.GetTx(ctx, event.Raw().TxHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eth tx")
	}

	// fixme solanaSeed routine is not really accurate, refactoring needed
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
		From:     srcItemIndex,
		To:       dstItemIndex,
	}

	if len(event.Bundle()) > 0 && notZero32(event.Salt()) {
		salt := event.Salt()
		msg.BundleSalt = hexutil.Encode(salt[:])

		msg.BundleData = hexutil.Encode(event.Bundle())
	}

	// that's not how generics should work, but for me, it looks better
	// like this than handling this corner-case in metadata package
	if !IsNFT(event.TokenParams().CoreTokenType) {
		return &msg, nil
	}

	knownItemMeta, err := m.tryFindMetaForItem(ctx, srcItemIndex)
	if err != nil {
		return nil, errors.Wrap(err, "unexpected err finding knowing meta for item")
	}

	shouldPopulateTokenID := IsNFT(event.TokenParams().CoreTokenType) && dstItemIndex.TokenID == ""

	solanaNetwork, err := m.tokenQueryClient.NetworkParams(ctx, &tokentypes.QueryNetworkParamsRequest{Name: SolanaChainName})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get network params", logan.F{
			"network": SolanaChainName,
		})
	}

	if knownItemMeta != nil {
		if shouldPopulateTokenID {
			if knownItemMeta.Seed == "" {
				return nil, errors.From(errors.New("seed is empty"), logan.F{
					"chain":         srcItemIndex.Chain,
					"contract_addr": srcItemIndex.Address,
					"id":            srcItemIndex.TokenID,
				})
			}

			dstItemIndex.TokenID = verifiers.MustGetPDA(solanaNetwork.Params.Contract, knownItemMeta.Seed)
		}

		return &msg, nil
	}

	meta, err := m.metadataProvider.EthItemMeta(ctx,
		event.TokenParams().ContractInterface,
		event.Token(),
		event.TokenId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get item meta")
	}

	solanaSeed, _ := verifiers.MustGenerateTokenSeed(dstItemIndex.Address) // TODO make it generated in ensuring indexes function
	if solanaSeed != "" && event.TokenParams().CoreTokenType == tokentypes.Type_ERC721 {
		meta.Seed = solanaSeed
		if shouldPopulateTokenID {
			dstItemIndex.TokenID = verifiers.MustGetPDA(solanaNetwork.Params.Contract, solanaSeed)
		}
	}

	msg.Meta = meta

	return &msg, nil
}

func (m *MessageMaker[T]) tryFindMetaForItem(ctx context.Context, associatedOnChainItem *tokentypes.OnChainItemIndex) (*tokentypes.ItemMetadata, error) {
	onChainItemResp, err := m.tokenQueryClient.OnChainItem(ctx, &tokentypes.QueryGetOnChainItemRequest{
		Chain:   associatedOnChainItem.Chain,
		Address: associatedOnChainItem.Address,
		TokenID: associatedOnChainItem.TokenID,
	})
	if err != nil {
		if res, ok := status.FromError(err); ok && res.Code() == codes.NotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "unexpected err while getting on-chain item")
	}

	itemResp, err := m.tokenQueryClient.Item(ctx, &tokentypes.QueryGetItemRequest{
		Index: onChainItemResp.Item.Item,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ")
	}

	return itemResp.Item.Meta, nil
}

func (m *MessageMaker[T]) itemOnChainIndices(ctx context.Context, e T) (
	*tokentypes.OnChainItemIndex,
	*tokentypes.OnChainItemIndex,
	error,
) {
	onChainItemSrc := e.OnChainItemIndex(m.homeChain)

	onChainItemDst, err := m.ensureDstItem(ctx, onChainItemSrc, e.Network())
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to ensure destination item", logan.F{
			"src_chain":         onChainItemSrc.Chain,
			"src_token_id":      onChainItemSrc.TokenID,
			"src_contract_addr": onChainItemSrc.Address,
			"dst_chain":         e.Network(),
		})
	}

	return onChainItemSrc, onChainItemDst, nil
}

func (m *MessageMaker[T]) ensureDstItem(ctx context.Context, srcOnChainItem *tokentypes.OnChainItemIndex, dstChain string) (*tokentypes.OnChainItemIndex, error) {
	// pushing luck trying to fetch on chain item index for destination chain
	dstItemResp, err := m.tokenQueryClient.OnChainItemByOther(ctx, &tokentypes.QueryGetOnChainItemByOtherRequest{
		Chain:       m.homeChain,
		Address:     srcOnChainItem.Address,
		TokenID:     srcOnChainItem.TokenID,
		TargetChain: dstChain,
	})
	if err != nil {
		if res, ok := status.FromError(err); ok && res.Code() != codes.NotFound {
			return nil, errors.Wrap(err, "unexpected err while getting dst on chain item")
		}
	}

	if dstItemResp != nil {
		return dstItemResp.Item.Index, nil
	}

	collectionResp, err := m.tokenQueryClient.CollectionByCollectionData(ctx, &tokentypes.QueryGetCollectionByCollectionDataRequest{
		Chain:   srcOnChainItem.Chain,
		Address: srcOnChainItem.Address,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get collection from core", logan.F{
			"chain": srcOnChainItem.Chain,
			"addr":  srcOnChainItem.Address,
		})
	}

	dstCollectionDataIndex := findCollectionDataIdxForChain(collectionResp.Collection, dstChain)
	if dstCollectionDataIndex == nil {
		return nil, errors.Wrap(verifiers.ErrWrongOperationContent, "no collection data for dst chain", logan.F{
			"collection": collectionResp.Collection.Index,
			"dst_chain":  dstChain,
		})
	}

	dstCollectionDataResp, err := m.tokenQueryClient.CollectionData(ctx, &tokentypes.QueryGetCollectionDataRequest{
		Chain:   dstCollectionDataIndex.Chain,
		Address: dstCollectionDataIndex.Address,
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to get collection data for dst chain", logan.F{
			"collection": collectionResp.Collection.Index,
			"dst_chain":  dstChain,
		})
	}

	if !IsNFT(dstCollectionDataResp.Data.TokenType) {
		return &tokentypes.OnChainItemIndex{
			Chain:   dstCollectionDataIndex.Chain,
			Address: dstCollectionDataIndex.Address,
		}, nil
	}

	// let the madness begin (c)

	// fetching collection's home chain data to get core-consistent tokenID for destination item
	collectionHomeChainData, err := m.getCollectionHomeChainData(ctx, collectionResp.Collection.Index)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get home collection data for source item")
	}

	targetChainTokenID := srcOnChainItem.TokenID

	// in case src item is on home chain we make item on destination chain have
	// same token ID as on source chain and take address from target chain
	if collectionHomeChainData.Index.Chain == srcOnChainItem.Chain {
		if dstCollectionDataIndex.Chain == SolanaChainName { // fixme refactoring of chain identification is needed
			targetChainTokenID = "" // TODO seed and token ID should be generated here
		}

		return &tokentypes.OnChainItemIndex{
			Chain:   dstCollectionDataIndex.Chain,
			Address: dstCollectionDataIndex.Address,
			TokenID: targetChainTokenID,
		}, nil
	}

	// in other case we get item's id from other chain
	// (assuming it does exist because it is guaranteed that
	// item has been already transferred to m.homeChain before)
	onChainItemFromDstCollectionResp, err := m.tokenQueryClient.OnChainItemByOther(ctx, &tokentypes.QueryGetOnChainItemByOtherRequest{
		Chain:       m.homeChain,
		Address:     srcOnChainItem.Address,
		TokenID:     srcOnChainItem.TokenID,
		TargetChain: collectionHomeChainData.Index.Chain,
	})

	if err != nil {
		if res, ok := status.FromError(err); ok && res.Code() != codes.NotFound {
			return nil, errors.Wrap(err, "unexpected err while getting dst on chain item")
		}
	}

	if onChainItemFromDstCollectionResp == nil {
		return nil, verifiers.ErrWrongOperationContent
	}

	return &tokentypes.OnChainItemIndex{
		Chain:   dstCollectionDataIndex.Chain,
		Address: dstCollectionDataIndex.Address,
		TokenID: onChainItemFromDstCollectionResp.Item.Index.TokenID,
	}, nil
}

// returns home chain CollectionData entity that is related to the collection that has chain and address on any other chain
func (m *MessageMaker[T]) getCollectionHomeChainData(ctx context.Context, collectionIndex string) (*tokentypes.CollectionData, error) {
	homeChainCollectionData, err := m.tokenQueryClient.NativeCollectionData(ctx,
		&tokentypes.QueryGetNativeCollectionDataRequest{
			Collection: collectionIndex,
		})
	if err != nil {
		return nil, errors.Wrap(err, "error fetching collection data for home chain")
	}

	return &homeChainCollectionData.Data, nil
}

func findCollectionDataIdxForChain(collection tokentypes.Collection, chain string) *tokentypes.CollectionDataIndex {
	for _, collectionData := range collection.Data {
		if collectionData.Chain == chain {
			return collectionData
		}
	}

	return nil
}

func IsNFT(t tokentypes.Type) bool {
	return t == tokentypes.Type_ERC721 || t == tokentypes.Type_ERC1155
}

func notZero32(bb [32]byte) bool {
	for _, b := range bb {
		if b != 0 {
			return true
		}
	}
	return false
}
