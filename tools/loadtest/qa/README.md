# Load test QA

Tooling for Load test 2.0: a load-test instance worked like a real one, for months, and judged all the time.

- [`driver/`](driver/): the activity driver. Runs the workload actions in `scenarios/` against a load-test
  instance at a customer profile's rates, and writes one JSON event per action.
- [`scenarios/`](scenarios/): the specs the driver runs, in the scenario catalogue's format. They hold start
  guesses only; the real rates come from a private profile file the driver reads at runtime.

The population generator and the other load-test QA scripts will move here too.
