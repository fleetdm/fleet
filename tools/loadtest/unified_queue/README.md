> [!NOTE]
> **Prefer [`dibble`](../../dibble/README.md) for seeding the software/script
> entities this loadtest needs.** Use:
> ```bash
> ./tools/dibble/dibble software   # seeds software titles
> ./tools/dibble/dibble scripts    # seeds saved scripts
> ```
> The actual install/enqueue-for-an-hour loadtest logic is unique to this tool;
> we'll consolidate when dibble's software upload path is finished.

# Load testing of the unified queue story

This is the Go program used to run load tests for the [unified queue story](https://github.com/fleetdm/fleet/issues/22866).

It expects some software to be available for install on both macOS and Windows (including VPP apps for macOS), and some scripts too, and it enqueues installs and script execution requests on every host in the Fleet deployment for an hour.
