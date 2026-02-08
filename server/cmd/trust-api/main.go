package main

import (
	"flag"
	"log"
	"net"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/trust"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	listenAddr := flag.String("listen", ":50051", "gRPC listen address")
	flag.Parse()

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *listenAddr, err)
	}

	srv, err := trust.NewServer()
	if err != nil {
		log.Fatalf("failed to init trust server: %v", err)
	}

	s := grpc.NewServer()
	baselineintegrityv1.RegisterTrustServiceServer(s, srv)
	reflection.Register(s)

	log.Printf("baselineintegrity-trust-api listening on %s", *listenAddr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server stopped: %v", err)
	}
}
