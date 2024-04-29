## fleetd-chrome 1.3.0 (Apr 29, 2024)

* Created a fix to recover after a rare RuntimeError coming from sqlite web assembly code by reinitializing the DB.

* Fixed a bug where values not derived from "actual" fleetd-chrome tables were not being displayed
  correctly (e.g., `SELECT 1` gets its value from the query itself, not a table)
