# Use a canary fleet to catch fleetd conflicts before they reach production

EDRs are suspicious by nature. They watch everything, and sometimes that means
flagging or blocking processes that are perfectly legitimate, including fleetd.
If you've ever had a Fleet agent silently stop reporting after a software update,
there's a decent chance your EDR had something to say about it.

The fix isn't to stop updating. It's to find out sooner, on machines you control,
before the problem lands on 5,000 devices.

This post walks through setting up a canary fleet in Fleet that runs the `edge`
update channel, so you catch compatibility issues early.

## What "edge" means

Fleet's agent (fleetd) ships three components: `orbit`, `osqueryd`, and Fleet
Desktop. Each one can be pinned to a specific version or channel in your agent
options.

Fleet supports two update channels (and also allows pinning to a specific version):

- `stable`: the current production release
- `edge`: the next release, available before it ships to stable
- A specific version string (e.g., `1.50.0`)

By putting a small group of devices on `edge`, you get early access to every
fleetd update. If something conflicts with CrowdStrike, Defender, your DLP tool,
or anything else in your stack, you'll see it on test hardware instead of in a
production incident.

> `update_channels` is only available in Fleet Premium.

## Set up your canary fleet

### 1. Create a fleet

In the Fleet UI, go to **Settings > Fleets** and create a new fleet. Call it
something obvious, like "Canary" or "Edge testers."

### 2. Configure agent options

In your fleet's agent options, set all three fleetd components to `edge`:

```yaml
agent_options:
  update_channels:
    orbit: edge
    osqueryd: edge
    desktop: edge
```

You can apply this via the Fleet UI under **Settings > Fleets > [your fleet] >
Agent options**, or with `fleetctl apply` if you're managing config as code.

### 3. Enroll test devices

Add at least one device for every platform your organization manages. If you
manage macOS, Windows, and Linux, you need a canary device for each. Platform
differences matter here. A conflict that only affects macOS won't show up on
a Windows test machine.

Good candidates for canary devices:

- IT team machines where the owner can report odd behavior
- Dedicated test hardware that runs your full security tool stack

The key requirement is that these devices run the same security software as your
production fleet. A canary device without your EDR installed isn't testing the
right thing.

## What to watch for

Once your canary fleet is running `edge`, keep an eye on:

- **Hosts going offline.** A device that stops checking in after a fleetd update
  is worth investigating. Check the fleetd logs on the device, then compare the
  timing against the update. The
  [Fleet troubleshooting guide](https://fleetdm.com/guides/fleet-troubleshooting-for-it-admins)
  has log locations for every platform.
- **EDR alerts.** If your EDR flags fleetd processes after an update, that's
  exactly the kind of early warning this setup is designed to surface.
- **Query failures.** If scheduled queries stop returning results, the osquery
  component may have a problem.

## One thing to know about downgrading

Once `orbit` has been upgraded to 1.20.0 or later, don't configure `orbit` to a
channel that contains a version older than 1.20.0. The auto-update system will
get into a restart loop. If you need to roll back, downgrade the channel itself
rather than changing the channel assignment. The
[agent configuration docs](https://fleetdm.com/docs/configuration/agent-configuration#update-channels)
cover this edge case in detail.

## Start small, catch problems early

A canary fleet costs almost nothing to set up and can save a lot of cleanup.
Pick one device per platform, point them at `edge`, make sure they're running
your real security stack, and then pay attention when fleetd updates roll
through. That's the whole setup.

If something breaks, you find out before it matters. If nothing breaks, your
production devices get a battle-tested update.

<meta name="articleTitle" value="Use a canary fleet to catch fleetd conflicts before they reach production">
<meta name="authorFullName" value="Kathy Satterlee">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-05-28">
<meta name="category" value="guides">
<meta name="description" value="Set up a canary fleet on Fleet's edge update channel to catch fleetd conflicts with your EDR and security stack before they reach production devices.">
