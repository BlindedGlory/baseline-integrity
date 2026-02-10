# Risk Scoring Model

This document defines the **privacy-safe risk scoring model** used by Baseline Integrity.

The goal of risk scoring is **not immediate detection**, but **longitudinal confidence assessment** across multiple matches.

No single signal is sufficient to trigger enforcement.

---

## Core Principles

Baseline Integrity risk scoring follows these principles:

- **Aggregate over inspect**  
  Only summary statistics are evaluated.

- **Accumulate over time**  
  Risk increases gradually across multiple observations.

- **Decay naturally**  
  Absence of suspicious behavior reduces accumulated risk.

- **Delay feedback**  
  Enforcement decisions are intentionally non-immediate.

This design reduces false positives and adversarial adaptation.

---

## Signal Types

Signals are derived from **server-authoritative observations**, not client instrumentation.

Examples include:
- impossible or highly improbable movement envelopes
- reaction time distributions outside expected variance
- statistically implausible consistency across matches
- violation frequency relative to match duration

Signals are always:
- schema-defined
- match-scoped
- aggregate-only

---

## Scoring Model (Conceptual)

Each match produces a set of **normalized risk contributions**:


Where:
- weights are configuration-specific
- normalization accounts for match duration and context
- raw values are never stored long-term

---

## Longitudinal Accumulation

Risk is accumulated across matches using bounded accumulation:


This ensures:
- short bursts do not dominate
- historical behavior matters
- risk does not grow unbounded

---

## Decay Behavior

Risk decays automatically over time:

- decay is continuous, not step-based
- decay rate is configurable per mode (casual vs ranked)
- clean play reduces risk without explicit forgiveness events

Decay is essential to:
- reduce false positives
- handle player improvement
- avoid permanent suspicion states

---

## Enforcement Thresholds

Thresholds are **not fixed constants**.

They may vary by:
- game mode
- trust tier
- population distribution
- operational policy

Thresholds are:
- server-controlled
- non-public
- adjustable without client updates

---

## Enforcement Ladder

Risk does not map directly to bans.

Typical actions include:
- increased server-side validation
- matchmaking segregation
- shadow analysis
- delayed review
- eventual enforcement (if sustained)

Immediate permanent bans are intentionally avoided.

---

## Adversarial Considerations

This model intentionally avoids:
- real-time feedback
- per-signal thresholds
- deterministic triggers

Attackers cannot reliably infer:
- which signals triggered risk
- when risk crossed a boundary
- how close they are to enforcement

---

## Summary

Baseline Integrity treats cheating as a **behavioral trend**, not an event.

By combining:
- server authority
- aggregate telemetry
- longitudinal analysis
- natural decay

the system reduces cheat effectiveness while preserving player privacy and platform trust.
