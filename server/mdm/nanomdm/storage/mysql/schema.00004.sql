ALTER TABLE command_results ADD COLUMN not_now_at TIMESTAMP NULL;
ALTER TABLE command_results ADD COLUMN not_now_tally INTEGER NOT NULL DEFAULT 0;
