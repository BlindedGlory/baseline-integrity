# Companion Attestation Claims Specification

This document defines the **allowed and forbidden attestation claims** for the Baseline Integrity Verified trust tier.

The companion app exists to provide **cryptographic claims**, not continuous monitoring or inspection.

---

## Design Principles

The companion app must adhere to the following principles:

- **Claims, not surveillance**  
  The companion reports boolean or bounded claims, not raw data.

- **Session-scoped**  
  All claims are bound to a specific session nonce.

- **Opt-in only**  
  Players must explicitly choose to participate in Verified mode.

- **No continuous monitoring**  
  The companion does not stream data or remain persistently active.

---

## Allowed Claim Categories

The following claim categories are permitted.

### Platform Integrity Claims

These claims assert properties of the execution environment without exposing raw system state.

Examples:
- Secure Boot enabled (claimed)
- TPM present (claimed)
- OS integrity enforcement enabled (claimed)

Claims are:
- boolean or enum-based
- platform-dependent
- explicitly declared in schema

---

### Cryptographic Binding Claims

These claims ensure that attestation is **bound to the active session**.

Examples:
- nonce binding confirmation
- attestation key ownership
- signature over session reference

These prevent replay and cross-session reuse.

---

### Companion Identity Claims

The companion may assert:
- possession of a per-install, rotatable key
- version compatibility
- supported feature set

The companion must **not** assert a stable hardware or device identifier.

---

## Forbidden Claims (Non-Negotiable)

The companion **must not** claim or report:

- process lists or memory contents
- running applications
- keystrokes, mouse input, or input timing
- screen contents or screenshots
- file system enumeration
- network traffic inspection
- unique or stable device fingerprints

If a claim would enable surveillance, it is out of scope.

---

## Claim Semantics

All claims are:

- declarative (true / false / bounded)
- cryptographically signed
- schema-defined
- auditable

Claims are evaluated **in aggregate** with server-side observations.
They are not treated as absolute proof.

---

## Failure and Degradation

If attestation fails or is unavailable:

- the session remains valid at the Open tier
- no punitive action is taken
- the player may continue play without Verified benefits

Verified tier is **additive**, not exclusionary.

---

## Security Considerations

This design intentionally limits:
- information leakage
- attack surface
- adversarial learning

The companion app does not attempt to prevent cheating directly.
It exists to **increase confidence**, not to enforce control.

---

## Summary

The Baseline Integrity companion app provides **bounded, opt-in, cryptographic claims** that strengthen trust without violating player privacy.

If a claim cannot be expressed without surveillance, it is explicitly forbidden.
