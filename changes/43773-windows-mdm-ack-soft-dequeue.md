- Reduced Windows MDM database load during bulk profile operations: acknowledged commands are now soft-dequeued
  (`acked_at`) in the ack transaction, pending-command checks use an index probe instead of per-row anti-joins, and the
  `has_pending_commands` flag is recomputed once per OMA-DM session instead of once per message.
