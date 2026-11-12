> [!NOTE]
> **Prefer [`dibble`](../../dibble/README.md) for seeding labels.** The equivalent is:
> ```bash
> ./tools/dibble/dibble labels --count N
> ```
> This tool is kept for backwards compatibility. We'll remove it once nothing references it.

# fleetd_labels

This tool can be used to set up a fixed set of manual labels to the hosts in a Fleet deployment.
This utility was used for load testing https://github.com/fleetdm/fleet/issues/13287 which required a set of labels applied to hosts.

The hardcoded numbers defined herein were agreed upon with the target customer.