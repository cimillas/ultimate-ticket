# ADR 0009: API versioning strategy

## Context
We need a clear versioning strategy before extracting services. The API is still
evolving, and we want to avoid the overhead of multiple versions while keeping
compatibility predictable.

## Decision
Use **implicit versioning** with strict changelog discipline:
- No breaking changes without an ADR + changelog entry.
- Additive changes only (new fields/endpoints) unless explicitly approved.
- Revisit versioning once external consumers require multiple versions.

## Alternatives considered
- `/v1` URL prefix: clearer contracts but higher maintenance and routing churn.
- Header-based versioning: flexible but adds complexity and tooling burden.

## Consequences
- The API surface must remain stable across iterations.
- Breaking changes must be rare and explicitly documented.
