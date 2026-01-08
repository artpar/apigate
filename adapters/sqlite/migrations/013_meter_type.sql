-- Add meter_type and estimated_cost_per_req to plans table
-- meter_type: "requests" (default) or "compute_units"
-- estimated_cost_per_req: estimated cost per request for pre-check (default 1.0)

ALTER TABLE plans ADD COLUMN meter_type TEXT NOT NULL DEFAULT 'requests';
ALTER TABLE plans ADD COLUMN estimated_cost_per_req REAL NOT NULL DEFAULT 1.0;
