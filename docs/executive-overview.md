# Baseline Integrity — Executive Overview

## Summary

Baseline Integrity is a **privacy-first competitive integrity platform** designed to reduce cheat advantage while avoiding invasive client control.

It replaces surveillance-based anti-cheat models with **server authority, cryptographic trust, and behavioral baselines**, allowing studios and platforms to enforce fairness without kernel drivers, process inspection, or continuous monitoring.

---

## The Problem

Modern anti-cheat systems increasingly rely on:
- kernel-level access
- broad process inspection
- opaque client surveillance
- high legal and ethical risk

These approaches:
- raise privacy and regulatory concerns
- fragment platform support (Linux, Steam Deck, cloud)
- increase operational and reputational risk
- create adversarial relationships with players

At the same time, cheating continues to adapt faster than invasive controls.

---

## The Baseline Integrity Approach

Baseline Integrity takes a different stance:

- **Server authority first**  
  Game servers remain the source of truth for movement, combat, and outcomes.

- **Evidence-based trust**  
  Player trust is accumulated over time using aggregate, privacy-safe signals.

- **Cryptographic verification**  
  Session trust is expressed via signed, verifiable tokens.

- **Optional stronger verification**  
  Players may opt into higher trust tiers for ranked or tournament environments.

---

## Privacy and Compliance Guarantees

Baseline Integrity enforces privacy at the protocol level:

- No kernel hooks by default
- No raw input capture
- No process or memory scanning
- No stable device identifiers
- No continuous surveillance

All telemetry is:
- aggregated
- schema-bound
- match-scoped
- auditable

If a signal is not defined in schema, it does not exist.

---

## Trust Model

Baseline Integrity uses **tiered trust**, not exclusionary enforcement:

- **Open tier**  
  Available to all players. Server-side validation and aggregate telemetry only.

- **Verified tier (opt-in)**  
  Companion-attested session tokens for higher-stakes play.

- **Tournament tier**  
  Narrow configuration, strongest validation, human review hooks.

Trust is **accumulative and decaying**, not instant and binary.

---

## Architectural Highlights

- Offline token verification using cached public keys
- Explicit key rotation and decentralization
- No runtime dependency on a central enforcement service
- Open protocols with breaking-change detection

This enables:
- low latency
- high availability
- distributed hosting
- independent verification

---

## Adoption Model

Baseline Integrity is designed for **incremental adoption**:

1. Server-side validation and telemetry (no client changes)
2. Offline trust token verification
3. Optional verified companion for ranked or competitive modes

No phase requires kernel drivers or invasive control.

---

## Strategic Value

Baseline Integrity aligns with:
- privacy regulation trends
- platform diversity (Linux, cloud, console)
- long-term player trust
- auditability and transparency requirements

It is intended to serve as a **reference architecture** for ethical competitive integrity, not a closed or opaque enforcement system.

---

## Positioning

Baseline Integrity does not attempt to “win” an arms race.

Instead, it aims to:
- raise the cost of cheating
- reduce false positives
- minimize player harm
- preserve platform trust

Competitive integrity should not require surveillance.
