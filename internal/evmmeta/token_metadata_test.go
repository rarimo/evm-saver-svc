package evmmeta

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rarimo/evm-saver-svc/pkg/ipfs"
	"github.com/rarimo/evm-saver-svc/pkg/metadata"
	"github.com/stretchr/testify/assert"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
)

func TestFetchTokenMeta(t *testing.T) {
	evmRPC := os.Getenv("EVM_RPC")
	if !assert.NotEmpty(t, evmRPC, "expected to be not empty") {
		return
	}

	ethClient, err := ethclient.Dial(evmRPC)
	if !assert.NoError(t, err, "expected to dial successfully") {
		return
	}

	metadataClient := metadata.NewClient(http.DefaultClient, ipfs.NewMockGateway(t), 10*time.Second)

	p := NewProvider(ethClient, metadataClient)

	ctx := context.Background()

	meta, err := p.TokenMetadata(ctx,
		gobind.ContractInterfaceERC1155,
		common.HexToAddress("0x9493e2970931D9a182AB99290B3F43347F1b8b9a"),
		big.NewInt(2))

	if !assert.NoError(t, err, "expected meta to be fetched successfully") {
		return
	}

	spew.Dump(meta)
}
