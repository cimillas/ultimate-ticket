# ADR 0001: Initial architecture and development approach

## Status
Accepted

## Date
2025-XX-XX

## Context
This project is a side-project aimed at learning by building a real-world,
high-scale ticketing system, starting from a toy prototype and evolving
incrementally.

Key constraints and goals at project start:
- Ticket sales for concert-like events.
- Inventory is **zone-based** (e.g. floor, stands, amphitheater).
- Users cannot select an exact seat; only quantity per zone.
- The system must prevent overselling.
- Development is **AI-assisted but human-guided**.
- **Test-Driven Development (TDD)** is mandatory.
- The system should remain understandable and evolvable.
- Early optimization and complex patterns should be avoided.

The project is expected to evolve over time to include:
- Waiting room / admission control
- Real payment integration
- Anti-abuse and bot mitigation
- Observability and SRE practices
- Potential scaling to very high concurrency

## Decision
We decide to:

1. **Start with a modular monolith**
   - A single Go service containing clearly separated modules
     (domain, storage, HTTP layer).
   - Internal boundaries must be clean to allow future service extraction.

2. **Use Go as the primary language**
   - Chosen for simplicity, performance, explicitness, and strong tooling.
   - Avoid advanced language features or clever abstractions early.

3. **Adopt strict Test-Driven Development (TDD)**
   - No production logic without tests.
   - Domain logic is tested first and thoroughly.
   - Bugs must be fixed via regression tests.

4. **Use Docker-first local development**
   - All dependencies (e.g. database) run via Docker Compose.
   - The system must be runnable locally with a single command.
   - No reliance on developer-specific local state.

5. **Use a relational database initially (PostgreSQL)**
   - Suitable for expressing and validating inventory invariants.
   - Simpler to reason about correctness in early stages.
   - Database choice may change later (e.g. DynamoDB for inventory),
     but only after core behavior is well understood.

6. **Avoid complex or “tricky” solutions initially**
   - No event buses, queues, sagas, or distributed locks at the start.
   - No premature concurrency or optimization.
   - Prefer boring, explicit, readable code.

7. **Defer non-essential concerns**
   - No payments, authentication, or waiting room in the initial milestone.
   - These will be introduced incrementally with separate ADRs.

8. **Require explicit approval for non-trivial decisions**
   - Any significant architectural or behavioral change must be proposed,
     discussed, and approved before implementation.
   - Such decisions must be recorded in subsequent ADRs.

## Consequences

### Positive
- Low cognitive load and fast iteration.
- Strong focus on correctness and domain understanding.
- Easier debugging and learning during early phases.
- Clean foundation for future scaling and refactoring.
- Clear guardrails for AI-assisted development.

### Negative
- Initial implementation will not be horizontally scalable.
- Some refactoring will be required when introducing distributed components.
- Certain performance characteristics will only be explored later.

These trade-offs are explicitly accepted as part of the learning-oriented,
incremental approach.

## Alternatives Considered

### Microservices from day one
Rejected because:
- High operational overhead.
- Harder to reason about correctness early.
- Slower feedback loop for a side-project.

### Serverless-first architecture
Rejected because:
- Harder to apply strict TDD.
- Harder to reason about inventory invariants.
- Increased cognitive load during early exploration.

### Event-driven architecture from the start
Rejected because:
- Introduces unnecessary complexity before domain behavior is well understood.
- Better introduced once core invariants are stable.

## Notes
This ADR establishes the baseline philosophy of the project.
All future architectural changes should reference this document and explicitly
state whether they extend, revise, or supersede its decisions.