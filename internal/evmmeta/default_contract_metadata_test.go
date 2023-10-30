//go:build manual_test
// +build manual_test

package evmmeta

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/contracts/evm-bridge/gobind"
)

func TestGetMetadata(t *testing.T) {
	ctx := context.Background()

	rpcurl := os.Getenv("RPC_URL")
	if !assert.NotEmpty(t, rpcurl, "RPC_URL env variable is not set") {
		return
	}

	client, err := ethclient.Dial(rpcurl)
	if !assert.NoError(t, err, "failed to connect to ethereum node") {
		return
	}

	t.Run("get erc20 metadata", func(t *testing.T) {
		addr := common.HexToAddress("0x40b5ea58c9Ec7521d6Ba42d90Af730ab55Ec720c")

		m, err := NewContractMetadata(addr, gobind.InterfaceIdERC20, client)
		if !assert.NoError(t, err, "failed to init metadata client") {
			return
		}

		name, err := m.Name(ctx)
		if !assert.NoError(t, err, "failed to get name") {
			return
		}

		symbol, err := m.Symbol(ctx)
		if !assert.NoError(t, err, "failed to get symbol") {
			return
		}

		uri, err := m.ContractURI(ctx)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Empty(t, uri, "contract uri expected to be empty") {
			return
		}

		_, err = m.TokenURI(ctx, big.NewInt(0))
		if !assert.Equal(t, ErrMethodNotSupported, errors.Cause(err), "should return not supported error") {
			return
		}

		fmt.Println(name)
		fmt.Println(symbol)
	})

	t.Run("get erc721 metadata", func(t *testing.T) {
		addr := common.HexToAddress("0x7fC0dC589B093fC7a3419161BAd677BE7616054C")

		m, err := NewContractMetadata(addr, gobind.InterfaceIdErc721, client)
		if !assert.NoError(t, err, "failed to init metadata client") {
			return
		}

		name, err := m.Name(ctx)
		if !assert.NoError(t, err, "failed to get name") {
			return
		}

		symbol, err := m.Symbol(ctx)
		if !assert.NoError(t, err, "failed to get symbol") {
			return
		}

		uri, err := m.ContractURI(ctx)
		if !assert.NoError(t, err, "failed to get contract uri") {
			return
		}

		tokenURI, err := m.TokenURI(ctx, big.NewInt(19))
		if !assert.NoError(t, err, "failed to get token uri") {
			return
		}

		fmt.Println(name)
		fmt.Println(symbol)
		fmt.Println(uri)
		fmt.Println(tokenURI)
	})

	t.Run("get erc1155 metadata", func(t *testing.T) {
		addr := common.HexToAddress("0x9493e2970931D9a182AB99290B3F43347F1b8b9a")
		m, err := NewContractMetadata(addr, gobind.InterfaceIdERC1155, client)
		if !assert.NoError(t, err, "failed to init metadata client") {
			return
		}

		tokenURI, err := m.TokenURI(ctx, big.NewInt(2))
		if !assert.NoError(t, err, "failed to get token uri") {
			return
		}

		fmt.Println(tokenURI)
	})
}
