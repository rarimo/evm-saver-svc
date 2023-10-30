package cli

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"

	"github.com/rarimo/evm-saver-svc/internal/services/grpc"
	"github.com/rarimo/evm-saver-svc/internal/services/voting"

	"github.com/rarimo/evm-saver-svc/internal/services/evm"

	"github.com/alecthomas/kingpin"
	"github.com/rarimo/evm-saver-svc/internal/config"
	"gitlab.com/distributed_lab/kit/kv"
)

func Run(args []string) bool {
	log := logan.New()

	defer func() {
		if rvr := recover(); rvr != nil {
			log.WithRecover(rvr).Error("app panicked")
		}
	}()

	cfg := config.New(kv.MustFromEnv())
	log = cfg.Log()

	app := kingpin.New("evm-saver-svc", "")

	runCmd := app.Command("run", "run command")

	allCmd := runCmd.Command("all", "run all services (evm listeners, evm voter, grpc api)")
	apiCmd := runCmd.Command("api", "run grpc api")
	voterCmd := runCmd.Command("voter", "run voter")
	saver := runCmd.Command("saver", "run saver")

	cmd, err := app.Parse(args[1:])
	if err != nil {
		log.WithError(err).Error("failed to parse arguments")
		return false
	}

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run := func(f func(ctx context.Context, cfg config.Config), name string) {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				cfg.Log().WithField("who", name).Info("finished routine")
			}()

			cfg.Log().WithField("who", name).Info("starting routine")
			f(ctx, cfg)
		}()
	}

	runSaver := func() {
		cfg.Log().Info("starting all savers")

		run(evm.RunIERC20Listener, "erc20-listener")
		run(evm.RunIERC721Listener, "erc721-listener")
		run(evm.RunIERC1155Listener, "erc1155-listener")
		run(evm.RunNativeListener, "native-listener")
	}

	runAll := func() {
		cfg.Log().Info("starting all services")

		run(voting.RunVoter, "voter")
		run(grpc.RunAPI, "grpc-api")
		runSaver()
	}

	if profiler := cfg.Profiler(); profiler.Enabled {
		profiler.RunProfiling()
	}

	switch cmd {
	case allCmd.FullCommand():
		runAll()
	case apiCmd.FullCommand():
		run(grpc.RunAPI, "grpc-api")
	case saver.FullCommand():
		runSaver()
	case voterCmd.FullCommand():
		run(voting.RunVoter, "voter")
	default:
		panic(errors.From(errors.New("unknown command"), logan.F{
			"raw_command": cmd,
		}))
	}

	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	wgch := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgch)
	}()

	select {
	case <-wgch:
		cfg.Log().Warn("all services stopped")
	case <-gracefulStop:
		cfg.Log().Info("received signal to stop")
		cancel()
		<-wgch
	}

	return true
}
