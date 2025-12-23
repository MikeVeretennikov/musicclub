package app

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/apsdehal/go-logger"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"musicclubbot/backend/internal/api"
	"musicclubbot/backend/internal/api/auth"
	"musicclubbot/backend/internal/config"
)

var propagatedCtxKeys = []string{"cfg", "log", "db"}

func Run(ctx context.Context) error {
	cfg := mustCfg(ctx)
	log := mustLog(ctx)

	lis, err := net.Listen("tcp", cfg.GRPCAddr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr(), err)
	}

	grpcServer := newGrpcServer(ctx)
	api.Register(grpcServer)
	reflection.Register(grpcServer)

	httpServer := &http.Server{
		Handler: newHTTPHandler(grpcServer),
	}

	go gracefulShutdown(ctx, grpcServer, httpServer)

	log.Infof("Starting gRPC server on %s", cfg.GRPCAddr())
	if err := httpServer.Serve(lis); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve gRPC/gRPC-Web: %w", err)
	}

	return nil
}

/* -------------------- helpers -------------------- */

func newGrpcServer(baseCtx context.Context) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			withBaseContext(baseCtx),
			loggingInterceptor,
			auth.AuthInterceptor,
		),
	)
}

func newHTTPHandler(grpcServer *grpc.Server) http.Handler {
	grpcWeb := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithOriginFunc(func(string) bool { return true }),
	)

	return h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if handlePreflight(w, r) {
				return
			}

			if isGrpcWebRequest(grpcWeb, r) {
				grpcWeb.ServeHTTP(w, r)
				return
			}

			http.NotFound(w, r)
		}),
		&http2.Server{},
	)
}

func gracefulShutdown(ctx context.Context, grpcServer *grpc.Server, httpServer *http.Server) {
	<-ctx.Done()
	grpcServer.GracefulStop()
	_ = httpServer.Shutdown(context.Background())
}

func withBaseContext(base context.Context) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		for _, key := range propagatedCtxKeys {
			if v := base.Value(key); v != nil {
				ctx = context.WithValue(ctx, key, v)
			}
		}
		return handler(ctx, req)
	}

}

func mustCfg(ctx context.Context) config.Config {
	return ctx.Value("cfg").(config.Config)
}

func mustLog(ctx context.Context) *logger.Logger {
	return ctx.Value("log").(*logger.Logger)
}

func handlePreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodOptions {
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Content-Type, X-Grpc-Web, X-User-Agent, Authorization",
	)
	w.WriteHeader(http.StatusNoContent)
	return true
}

func isGrpcWebRequest(gw *grpcweb.WrappedGrpcServer, r *http.Request) bool {
	return gw.IsGrpcWebRequest(r) ||
		gw.IsGrpcWebSocketRequest(r) ||
		gw.IsAcceptableGrpcCorsRequest(r)
}
