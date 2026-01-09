# Domain concepts

This project models zone-based ticket inventory. Below is a quick glossary.

## Event
A ticketed experience (concert, show, match). An event groups zones and defines
when it happens (`starts_at`). An event is marked complete when all its zones
have zero availability. Events can also be cancelled; cancellations set a
`cancelled_at` timestamp and block further holds or confirmations. Once the
start time passes, events are marked `closed` and reject new zones, holds, or
confirmations.
Closing does not refund orders; cancellations do.

## Zone
A sellable area within an event (e.g., floor, stands). Each zone has a capacity
(number of tickets that can be sold) and is the unit of inventory. Availability
is derived from capacity minus confirmed orders and active (unexpired) holds.

## Hold
A temporary reservation of `quantity` tickets in a zone. Holds have a TTL
(`expires_at`) and prevent overselling while a customer completes checkout.
Holds are owned by the authenticated user and are created with an idempotency
key scoped to that user. Holds are blocked once an event is closed or cancelled.
Cancelled events invalidate active holds.

## Confirmation (Order)
A confirmation turns an active hold into a finalized purchase. It is idempotent
and returns an order record. If a hold is expired or already confirmed, the
confirmation fails. Holds cannot be confirmed once an event is closed; those
holds are marked invalid. If an event is cancelled, confirmed orders are marked
as refunded.

## User
Users can register and log in to create holds and confirm them. There are two
roles: `user` (standard) and `admin` (full access to admin tooling). Admin
accounts are provisioned via backend configuration.

## Session
Sessions are stored in the database and tracked via an httpOnly cookie. Sessions
expire after a TTL and are refreshed on activity (sliding expiration). Invalid
or expired sessions reject protected endpoints.

## Typical flow
1. Create an event.
2. Create one or more zones for the event.
3. Create a hold for a zone.
4. Confirm the hold to finalize the order.
