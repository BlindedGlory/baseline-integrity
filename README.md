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

## Offline TierToken Verification (Game Servers)

Baseline Integrity is designed so **game servers do not need to call the Trust API on every request**.
Instead, servers can **verify TierTokens offline** using cached public keys.

This reduces latency, avoids centralized enforcement, and keeps trust **cryptographic and auditable**.

### Overview

1. Game server periodically fetches public signing keys from the Trust API
2. Keys are cached using `cache_until`
3. Incoming TierTokens are verified locally:
   - canonical payload
   - wrapper ↔ payload binding
   - expiration
   - Ed25519 signature

No kernel hooks. No device fingerprinting. No spyware.

---

### Fetch and cache public keys

```go
// Call TrustService.GetPublicKeys periodically (e.g. on boot and before cache expiration).
resp, err := trustClient.GetPublicKeys(ctx, &baselineintegrityv1.GetPublicKeysRequest{
	Purpose: "tier_tokens",
})
if err != nil {
	log.Fatalf("fetch public keys: %v", err)
}

keys := make(verify.PublicKeySet)
for _, k := range resp.Keys {
	keys[k.KeyId] = k.Ed25519 // 32-byte Ed25519 public key
}


## Getting started
### Prerequisites
- `buf` installed

### Lint + generate code
```bash
buf lint
buf generate
