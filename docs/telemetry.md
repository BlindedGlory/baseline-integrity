# Telemetry Principles

Telemetry in Baseline Integrity is:

- Aggregate-only (no per-frame streams)
- Feature-based (counters, histograms, quantiles)
- Match-scoped
- Server-authoritative

## Examples of valid telemetry
- envelope violations
- reaction time distributions
- impossible movement counts

## Examples of invalid telemetry
- keystrokes
- mouse paths
- raw packet logs
- process names

Telemetry exists to support **trend analysis**, not surveillance.
