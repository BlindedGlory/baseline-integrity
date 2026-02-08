package telemetry

import (
	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
)

// Server is a minimal TelemetryService skeleton.
// Embedding UnimplementedTelemetryServiceServer provides default Unimplemented responses.
type Server struct {
	baselineintegrityv1.UnimplementedTelemetryServiceServer
}
