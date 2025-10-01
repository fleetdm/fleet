## fleetd-chrome 1.3.3 (August 8, 2025)

* Fixed a bug which caused fleetd-chrome to fail enrollment.

## fleetd-chrome 1.3.2 (Feb 28, 2025)

- Fixed "privacy_preferences" table query to return results correctly.

## fleetd-chrome 1.3.1 (May 20, 2024)

* Fixed bug where fleetd-chrome sent multiple read requests to Fleet server at the same time.

* Improved console log output messages when Fleet server is down.

## fleetd-chrome 1.3.0 (Apr 29, 2024)

* Created a fix to recover after a rare RuntimeError coming from sqlite web assembly code by reinitializing the DB.

* Fixed a bug where values not derived from "actual" fleetd-chrome tables were not being displayed
  correctly (e.g., `SELECT 1` gets its value from the query itself, not a table)
