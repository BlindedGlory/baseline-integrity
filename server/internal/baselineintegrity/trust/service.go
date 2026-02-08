package trust

import (
	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
)

// Server is a minimal TrustService skeleton.
// Embedding UnimplementedTrustServiceServer provides default Unimplemented responses.
type Server struct {
	baselineintegrityv1.UnimplementedTrustServiceServer
}
