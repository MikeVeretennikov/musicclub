package app

import (
	"context"
	"fmt"
	"net"

	"github.com/apsdehal/go-logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"musicclubbot/backend/internal/api"
	"musicclubbot/backend/internal/config"
)

// Run initializes and starts the gRPC server with stub handlers.
func Run(ctx context.Context) error {
	cfg := ctx.Value("cfg").(config.Config)
	log := ctx.Value("log").(*logger.Logger)
	lis, err := net.Listen("tcp", cfg.GRPCAddr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr(), err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(loggingInterceptor))

	api.Register(grpcServer)
	reflection.Register(grpcServer)

	// Graceful stop on context cancellation.
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Infof("Starting gRPC server on %s", cfg.GRPCAddr())
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve gRPC: %w", err)
	}

	return nil
}
