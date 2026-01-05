-- Allow invalid holds after an event starts.
ALTER TABLE holds DROP CONSTRAINT IF EXISTS holds_status_check;
ALTER TABLE holds ADD CONSTRAINT holds_status_check
  CHECK (status IN ('active', 'confirmed', 'expired', 'invalid'));
