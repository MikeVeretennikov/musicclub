package app

import (
	"context"
	"time"

	"github.com/apsdehal/go-logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func loggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	log := ctx.Value("log").(*logger.Logger)

	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	ip := realIPFromContext(ctx)

	if err != nil {
		if ip != "" {
			log.Errorf("[%s] %s failed: %v", ip, info.FullMethod, err)
		} else {
			log.Errorf("%s failed: %v", info.FullMethod, err)
		}
		return resp, err
	}

	if ip != "" {
		log.Infof("[%s] %s handled in %s", ip, info.FullMethod, duration)
	} else {
		log.Infof("%s handled in %s", info.FullMethod, duration)
	}

	return resp, nil
}

func realIPFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// gRPC metadata keys are lowercase
	if values := md.Get("x-real-ip"); len(values) > 0 {
		return values[0]
	}

	return ""
}
