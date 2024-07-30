ALTER TABLE enrollment_queue ADD INDEX (priority DESC, created_at);
