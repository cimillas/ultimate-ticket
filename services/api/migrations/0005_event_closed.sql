-- Allow events to transition to closed once starts_at has passed.
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_status_check;
ALTER TABLE events ADD CONSTRAINT events_status_check
  CHECK (status IN ('active', 'closed', 'cancelled'));
