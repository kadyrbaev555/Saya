package platform

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	hp "google.golang.org/grpc/health/grpc_health_v1"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Context() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
func DB(ctx context.Context, dsn string) (*sqlx.DB, error) {
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	db, e := sqlx.ConnectContext(c, "postgres", dsn)
	if e == nil {
		db.SetMaxOpenConns(10)
	}
	return db, e
}
func Serve(ctx context.Context, register func(*grpc.Server)) error {
	s := grpc.NewServer()
	register(s)
	h := health.NewServer()
	hp.RegisterHealthServer(s, h)
	h.SetServingStatus("", hp.HealthCheckResponse_SERVING)
	l, e := net.Listen("tcp", ":9000")
	if e != nil {
		return e
	}
	go func() {
		<-ctx.Done()
		h.Shutdown()
		done := make(chan struct{})
		go func() { s.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.Stop()
		}
	}()
	e = s.Serve(l)
	if errors.Is(e, grpc.ErrServerStopped) {
		return nil
	}
	return e
}
func Loop(ctx context.Context, fn func(context.Context) error) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				task, cancel := context.WithTimeout(ctx, 10*time.Second)
				e := fn(task)
				cancel()
				if e != nil {
					slog.Warn("background delivery pending")
				}
			}
		}
	}()
}
