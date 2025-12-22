package app

import (
	"context"

	"github.com/apsdehal/go-logger"
	"google.golang.org/grpc"
)

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log := ctx.Value("log").(*logger.Logger)
	log.Infof("Received request for %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Errorf("Error handling %s: %v", info.FullMethod, err)
	} else {
		log.Infof("Successfully handled %s", info.FullMethod)
	}
	return resp, err
}
