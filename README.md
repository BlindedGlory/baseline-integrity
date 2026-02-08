# Baseline Integrity

## Design stance

Baseline Integrity is intentionally conservative in scope.
It prioritizes competitive fairness and user privacy over invasive or opaque enforcement techniques.

**Privacy-first competitive integrity** through **server authority**, **behavioral baselines**, and **cryptographic trust**.

Baseline Integrity is a cross-platform integrity platform designed to reduce cheat advantage and raise the cost of cheating **without invasive client control**. It treats trust as evidence-based and tiered: anyone can play, and players can optionally opt into stronger verification for ranked/tournament environments.

## What this is
- **Server-authoritative** integrity support (movement/combat validation lives server-side)
- **Aggregated telemetry** + multi-match risk scoring (no raw input capture)
- **Opt-in Verified mode** via a separate companion app (no kernel hooks by default)
- **Open-source, auditable protocols** and privacy boundaries enforced by schema

## What this is NOT
- Not kernel spyware
- Not process/memory scanning
- Not keystroke logging, screen capture, or “inspect everything” telemetry
- Not single-signal instant bans (risk is accumulated and decays over time)

## Trust tiers
- **Open**: server-side scoring only
- **Verified** (opt-in): companion-attested session token (TPM/Secure Boot/other claims as supported)
- **Tournament**: narrow configuration, strongest validation, review hooks

## Architecture (high level)
- **Game Server**: authoritative validation + produces privacy-safe match aggregates
- **Trust API**: session nonces, policy, tier token minting/verification
- **Telemetry Ingest**: accepts aggregated match features (server-signed)
- **Scoring/Enforcement**: multi-match risk scoring + enforcement ladder (delayed feedback)

## Repository layout
- `proto/` — Protobuf protocol definitions (buf-managed)
- `server/` — Trust API, telemetry ingest, scoring worker (Go)
- `companion/` — Verified companion app (Rust)
- `sdk/` — Unity/Unreal/C-ABI SDKs (thin integration layers)
- `docs/` — threat model, privacy spec, telemetry feature dictionary

## Getting started
### Prerequisites
- `buf` installed

### Lint + generate code
```bash
buf lint
buf generate
