# Baseline Integrity

## Design stance

Baseline Integrity is intentionally conservative in scope.
It prioritizes competitive fairness and user privacy over invasive or opaque enforcement techniques.

**Privacy-first competitive integrity** through **server authority**, **behavioral baselines**, and **cryptographic trust**.

Baseline Integrity is a cross-platform integrity platform designed to reduce cheat advantage and raise the cost of cheating **without invasive client control**. It treats trust as evidence-based and tiered: anyone can play, and players can optionally opt into stronger verification for ranked or tournament environments.

---

## Privacy Guarantees

Baseline Integrity enforces privacy at the protocol level:

- No kernel hooks by default
- No raw input capture
- No process lists or memory inspection
- No stable device identifiers
- No continuous surveillance

All client-reported data is:
- aggregated
- schema-bound
- purpose-limited
- verifiable

**If a signal is not defined in schema, it does not exist.**

---

## What this is

- **Server-authoritative** integrity support (movement/combat validation lives server-side)
- **Aggregated telemetry** + multi-match risk scoring (no raw input capture)
- **Opt-in Verified mode** via a separate companion app
- **Open-source, auditable protocols** with privacy boundaries enforced by schema

## What this is NOT

- Not kernel spyware
- Not process or memory scanning
- Not keystroke logging, screen capture, or “inspect everything” telemetry
- Not single-signal instant bans (risk accumulates and decays over time)

---

## Trust tiers

- **Open**: server-side scoring only
- **Verified** (opt-in): companion-attested session token (TPM / Secure Boot / supported claims)
- **Tournament**: narrow configuration, strongest validation, review hooks

---

## Architecture (high level)

- **Game Server**: authoritative validation + produces privacy-safe match aggregates
- **Trust API**: session nonces, policy, tier token minting and verification
- **Telemetry Ingest**: accepts aggregated match features (server-signed)
- **Scoring / Enforcement**: multi-match risk scoring with delayed enforcement ladder

---

## Repository layout

- `proto/` — Protobuf protocol definitions (buf-managed)
- `server/` — Trust API, telemetry ingest, scoring worker (Go)
- `companion/` — Verified companion app (Rust)
- `sdk/` — Unity / Unreal / C-ABI SDKs (thin integration layers)
- `docs/` — threat model, privacy spec, telemetry feature dictionary

See [`docs/telemetry.md`](docs/telemetry.md) for telemetry constraints and guarantees.

---

## Governance & Evolution

- Protocols are versioned (`baselineintegrity.v1`)
- Breaking changes are detected via `buf`
- Generated code is never committed
- New telemetry signals require schema changes
- Privacy boundaries cannot be bypassed by implementation

Baseline Integrity favors slow, auditable evolution over rapid opaque changes.

---

## Adoption Path

Studios can adopt Baseline Integrity incrementally:

1. **Phase 1 — Open tier only**
   - Server-side validation
   - Aggregated telemetry
   - No client changes required

2. **Phase 2 — Offline token verification**
   - Trust API integration
   - Cached public keys
   - No runtime dependency

3. **Phase 3 — Verified tier (opt-in)**
   - Companion app
   - Cryptographic attestation
   - Ranked / tournament use

No phase requires kernel drivers or invasive control.

---

## Why Baseline Integrity Exists

Competitive integrity should not require surveillance.

Players deserve fairness **and** privacy.  
Studios deserve integrity **without** legal or ethical risk.  
Platforms deserve systems that can be audited, reasoned about, and trusted.

Baseline Integrity is an attempt to raise the bar — technically and ethically.

---

## Offline TierToken Verification (Game Servers)

Baseline Integrity is designed so **game servers do not need to call the Trust API on every request**.
Instead, servers can **verify TierTokens offline** using cached public keys.

This reduces latency, avoids centralized enforcement, and keeps trust **cryptographic and auditable**.

### Overview

1. Game servers periodically fetch public signing keys from the Trust API
2. Keys are cached using `cache_until`
3. Incoming TierTokens are verified locally:
   - canonical payload
   - wrapper ↔ payload binding
   - expiration
   - Ed25519 signature

No kernel hooks. No device fingerprinting. No spyware.

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
