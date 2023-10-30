package grpc

import (
	"context"
	"net"

	"github.com/rarimo/evm-saver-svc/internal/config"
	"github.com/rarimo/evm-saver-svc/internal/services/voting"
	"gitlab.com/distributed_lab/logan/v3"
	rarimotypes "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	lib "gitlab.com/rarimo/savers/saver-grpc-lib/grpc"
	"gitlab.com/rarimo/savers/saver-grpc-lib/voter"
	"gitlab.com/rarimo/savers/saver-grpc-lib/voter/verifiers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type saverService struct {
	lib.UnimplementedSaverServer
	log      *logan.Entry
	voter    *voter.Voter
	rarimo   *grpc.ClientConn
	listener net.Listener
}

func RunAPI(ctx context.Context, cfg config.Config) {
	cfg.Log().Info("starting grpc api")

	srv := grpc.NewServer()

	lib.RegisterSaverServer(srv, &saverService{
		log:      cfg.Log(),
		rarimo:   cfg.Cosmos(),
		listener: cfg.Listener(),
		voter: voter.NewVoter(
			cfg.Ethereum().NetworkName,
			cfg.Log().WithField("who", "evm-saver-voter"),
			cfg.Broadcaster(),
			map[rarimotypes.OpType]voter.Verifier{
				rarimotypes.OpType_TRANSFER: verifiers.NewTransferVerifier(
					voting.NewTransfersVerifier(cfg),
					cfg.Log(),
				),
			},
		),
	})

	serve(ctx, srv, cfg)
}

// gRPC service implementation

var _ lib.SaverServer = &saverService{}

func (s *saverService) Revote(ctx context.Context, req *lib.RevoteRequest) (*lib.RevoteResponse, error) {
	op, err := rarimotypes.NewQueryClient(s.rarimo).Operation(ctx, &rarimotypes.QueryGetOperationRequest{Index: req.Operation})
	if err != nil {
		s.log.WithError(err).Error("error fetching op")
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if err := s.voter.Process(ctx, op.Operation); err != nil {
		s.log.WithError(err).Error("error processing op")
		return nil, status.Error(codes.Internal, "Internal error")
	}

	return &lib.RevoteResponse{}, nil
}