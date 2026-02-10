# Platform & Legal Readiness Checklist

This checklist is intended for **platform policy, legal, security, and compliance review** of the Baseline Integrity system.

Each item represents an explicit design guarantee.

---

## Client-Side Behavior

- [ ] No kernel drivers are required by default
- [ ] No process or memory scanning is performed
- [ ] No raw input (keyboard/mouse/controller) is captured
- [ ] No screen capture or video capture occurs
- [ ] No stable or unique device identifiers are collected
- [ ] No continuous background monitoring is performed

---

## Telemetry Constraints

- [ ] Telemetry is aggregate-only
- [ ] Telemetry is match-scoped
- [ ] Telemetry schema is explicit and versioned
- [ ] New telemetry signals require schema changes
- [ ] Telemetry cannot exceed defined counters, histograms, or quantiles
- [ ] Raw logs, packet traces, or streams are out of scope

See [`docs/telemetry.md`](telemetry.md) for full constraints.

---

## Trust & Verification

- [ ] Trust is tiered and opt-in
- [ ] All players may participate at the Open tier
- [ ] Verified tier requires explicit player consent
- [ ] Attestation is claims-based, not surveillance-based
- [ ] Attestation claims are schema-defined and auditable
- [ ] Failure to attest does not result in exclusion or punishment

See [`docs/companion-attestation.md`](companion-attestation.md).

---

## Cryptographic Guarantees

- [ ] Session trust is expressed via signed TierTokens
- [ ] Tokens are verifiable offline by game servers
- [ ] Public signing keys are discoverable and cacheable
- [ ] Key rotation does not break verification
- [ ] No runtime dependency on a central enforcement service

---

## Enforcement Model

- [ ] No single signal triggers enforcement
- [ ] Risk is accumulated across multiple matches
- [ ] Risk decays naturally over time
- [ ] Enforcement is delayed and graduated
- [ ] Immediate permanent bans are avoided by design

See [`docs/risk-scoring.md`](risk-scoring.md).

---

## Governance & Change Control

- [ ] Protocols are versioned
- [ ] Breaking changes are detected automatically
- [ ] Generated code is not committed
- [ ] Privacy boundaries cannot be bypassed by implementation changes
- [ ] Documentation defines normative behavior

---

## Regulatory Alignment

- [ ] Designed to minimize personal data collection
- [ ] Supports GDPR-style data minimization principles
- [ ] Avoids persistent identifiers
- [ ] Supports platform diversity (Linux, cloud, console)

---

## Summary

Baseline Integrity is designed to enable **competitive integrity without surveillance**.

This checklist exists to make review explicit, auditable, and repeatable.
