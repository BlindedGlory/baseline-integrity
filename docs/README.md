# Baseline Integrity — Specifications

This directory contains the **normative specifications and guarantees** for the Baseline Integrity platform.

These documents define:
- threat boundaries
- privacy constraints
- telemetry limits
- trust and verification contracts

They are intended for:
- platform reviewers
- security teams
- legal/compliance stakeholders
- engine and SDK integrators

Implementation **must conform** to these specifications.

---

## Document Index

### Core Principles
- [`threat-model.md`](threat-model.md)  
  Defines adversary scope, assumptions, and explicit non-goals.

- [`telemetry.md`](telemetry.md)  
  Defines what telemetry may and may not exist, and why.

- [`risk-scoring.md`](risk-scoring.md)  
  Defines privacy-safe, longitudinal risk accumulation and decay.


### Trust & Verification
- TierToken format and signing (see `proto/baselineintegrity/v1/trust.proto`)
- Offline verification contract (see README: *Offline TierToken Verification*)

- [`companion-attestation.md`](companion-attestation.md)  
  Defines allowed and forbidden claims for the Verified trust tier.


### Governance
- Protocol versioning (`baselineintegrity.v1`)
- Schema-enforced privacy boundaries
- Breaking-change detection via `buf`

- [`platform-readiness.md`](platform-readiness.md)  
  Platform, legal, and compliance review checklist.

- [`whitepaper-outline.md`](whitepaper-outline.md)  
  RFC-style outline for external publication and platform review.


---

## Design Philosophy (Non-Negotiable)

- Server authority over client control
- Evidence-based trust, not surveillance
- Aggregation over inspection
- Transparency over obscurity

If a behavior is not specified here or in protocol schema, it is **out of scope by design**.
