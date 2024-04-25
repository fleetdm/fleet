## fleetd-chrome 1.3.0 (Apr 25, 2024)

* Reinitialize DB and recover after a rare RuntimeError coming from sqlite web assembly code.

* Fix a bug where values not derived from "actual" fleetd-chrome tables were not being displayed
  correctly (e.g., `SELECT 1` gets its value from the query itself, not a table)
