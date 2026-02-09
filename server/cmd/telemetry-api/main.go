package main

import (
	"flag"
	"log"
	"net"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	listenAddr := flag.String("listen", ":50052", "gRPC listen address")
	flag.Parse()

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *listenAddr, err)
	}

	// ✅ Construct telemetry server properly (sets sinkDir)
	srv, err := telemetry.NewServer()
	if err != nil {
		log.Fatalf("failed to init telemetry server: %v", err)
	}

	s := grpc.NewServer()
	baselineintegrityv1.RegisterTelemetryServiceServer(s, srv)
	reflection.Register(s)

	log.Printf("baselineintegrity-telemetry-api listening on %s", *listenAddr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server stopped: %v", err)
	}
}
