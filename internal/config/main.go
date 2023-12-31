package config

import (
	"github.com/rarimo/saver-grpc-lib/broadcaster"
	"github.com/rarimo/saver-grpc-lib/metrics"
	"github.com/rarimo/saver-grpc-lib/voter"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"google.golang.org/grpc"
)

type Config interface {
	comfig.Logger
	comfig.Listenerer
	broadcaster.Broadcasterer
	voter.Subscriberer
	metrics.Profilerer

	Ethereum() *Ethereum
	Cosmos() *grpc.ClientConn
	Tendermint() *http.HTTP
}

type config struct {
	comfig.Logger
	comfig.Listenerer
	broadcaster.Broadcasterer
	voter.Subscriberer
	metrics.Profilerer

	ethereum   comfig.Once
	cosmos     comfig.Once
	tendermint comfig.Once

	getter kv.Getter
}

func New(getter kv.Getter) Config {
	return &config{
		getter:        getter,
		Logger:        comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Listenerer:    comfig.NewListenerer(getter),
		Broadcasterer: broadcaster.New(getter),
		Subscriberer:  voter.NewSubscriberer(getter),
		Profilerer:    metrics.New(getter),
	}
}
