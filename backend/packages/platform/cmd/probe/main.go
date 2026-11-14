package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	hp "google.golang.org/grpc/health/grpc_health_v1"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok := false
	if len(os.Args) > 1 && os.Args[1] == "http" {
		req, e := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:8080/health/ready", nil)
		if e == nil {
			response, e := http.DefaultClient.Do(req)
			if e == nil {
				ok = response.StatusCode == 200
				response.Body.Close()
			}
		}
	} else {
		c, e := grpc.NewClient("127.0.0.1:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if e == nil {
			defer c.Close()
			v, e := hp.NewHealthClient(c).Check(ctx, &hp.HealthCheckRequest{})
			ok = e == nil && v.Status == hp.HealthCheckResponse_SERVING
		}
	}
	if !ok {
		os.Exit(1)
	}
}
