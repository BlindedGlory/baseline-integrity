# Phase 3A — Step 3B: Risk Worker

## Goal
Implement a standalone worker that consumes finalized telemetry events from the filesystem outbox, computes deterministic risk contributions, and persists per-player RiskState safely and idempotently.

This completes the end-to-end flow:
Telemetry → Outbox → Risk Scoring → Persistent Risk State

---

## Non-Goals
- No bans or enforcement
- No trust tier changes
- No client-side logic
- No heuristics outside config
- No real-time decisions

---

## Inputs
- Outbox events: `match_finalized`
- Telemetry aggregates:
  `.baselineintegrity/telemetry/match_<match_id>.jsonl`
- Existing RiskState (if present)

## Outputs
- Updated per-player RiskState:
  `.baselineintegrity/risk/players/<player_id>.json`
- Acknowledged or failed outbox events

---

## New Binaries
