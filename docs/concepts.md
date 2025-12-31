# Domain concepts

This project models zone-based ticket inventory. Below is a quick glossary.

## Event
A ticketed experience (concert, show, match). An event groups zones and defines
when it happens (`starts_at`).

## Zone
A sellable area within an event (e.g., floor, stands). Each zone has a capacity
(number of tickets that can be sold) and is the unit of inventory.

## Hold
A temporary reservation of `quantity` tickets in a zone. Holds have a TTL
(`expires_at`) and prevent overselling while a customer completes checkout.
Holds are created with an idempotency key.

## Confirmation (Order)
A confirmation turns an active hold into a finalized purchase. It is idempotent
and returns an order record. If a hold is expired or already confirmed, the
confirmation fails.

## Typical flow
1. Create an event.
2. Create one or more zones for the event.
3. Create a hold for a zone.
4. Confirm the hold to finalize the order.
