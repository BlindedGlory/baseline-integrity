# Baseline Integrity — Phase Status

## Project
Baseline Integrity  
Privacy-first competitive integrity through server authority, behavioral baselines, and cryptographic trust.

Repo: https://github.com/BlindedGlory/baseline-integrity

---

## Locked Principles
- No kernel drivers by default
- No process scanning or raw input capture
- No stable device fingerprinting
- Server-authoritative integrity
- Cryptographic trust (Ed25519)
- Tiered trust model (Open / Verified / Tournament)
- Open-source, auditable, platform-safe

---

## Current Phase
**Phase 3A — Risk Scoring Pipeline**

### Completed
- Trust API (production-ready)
- Telemetry ingest API (schema-guarded, privacy-safe)
- Filesystem outbox (durable, atomic, replayable)
- Risk model (longitudinal, decay-based)
- Telemetry → MatchRisk mapping (deterministic, config-driven)

### Current Step
**Step 3B — Risk Worker (next to implement)**

Purpose:
Consume finalized telemetry via outbox, compute deterministic risk, and persist per-player RiskState.

---

## What’s Next (Step 3B)
- Implement standalone risk worker binary
- Claim outbox `match_finalized` events
- Load telemetry match aggregates
- Compute per-player MatchRisk
- Apply decay + accumulation
- Persist RiskState atomically
- Guarantee idempotency and replay safety

Detailed implementation plan:
→ `docs/phase-3a-step-3b-risk-worker.md`

---

## After Step 3B
- Step 4: Idempotency hardening & replay reaper
- Step 5: Stable, versioned RiskState schema
- Step 6: Enforcement ladder (non-authoritarian)
- Step 7: Audit & explainability

---

## Status Summary
🟢 Trust & Telemetry: Production-ready  
🟢 Outbox: Integrated & stable  
🟡 Risk pipeline: Worker pending (Step 3B)
