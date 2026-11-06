# Microsoft's October update closed 1,100 CVEs. Which of your devices are still waiting?

*Microsoft's own diagnostic data shows most enterprise devices lag weeks behind a big patch release. The number that matters for your team is the one measured on your own hosts.*

## Key takeaways

- **A shipped patch isn't an applied patch.** Microsoft says its October update closed more than 1,100 CVEs, yet its diagnostic data shows fewer than half of enterprise devices had installed it after two weeks. Exposure lives in the gap between release day and install day.

- **About a quarter of devices were still behind at four weeks.** A month after release, a meaningful share of the enterprise population was still running the old build. Every one of those devices is open to anything the update fixed.

- **Vendor telemetry describes everyone, not you.** An aggregate figure can't tell you whether your laptops, servers, or contractor machines are in the lagging group. Only your own inventory can.

- **You can measure the same gap on your own hosts.** Fleet's agent reports what each device has installed, so a team can count how many hosts took an update at 14 days and at 28 days, across Windows, macOS, and Linux.

- **Lagging devices need an owner and a deadline.** A list of stragglers is only useful if each device has a person responsible and a date to check again.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See patch status across your devices</a>

Microsoft's October update closed more than 1,100 CVEs. Microsoft's own diagnostic data, as reported in its Windows blog post, also shows that fewer than half of enterprise devices had installed the update within two weeks, and about a quarter still hadn't after four weeks.

Those are fleet-wide averages, and averages hide where your risk sits. The better question is how your own devices compare, and how you would find out.

## Why the install gap matters more than the CVE count

A large CVE count makes for a good headline, but a vulnerability is only closed on the devices that run the fix. For the two to four weeks after a release, a device that hasn't updated is exactly as exposed as it was the day before. Attackers know the fix is public, so the window after disclosure is when an unpatched device is easiest to target.

## Why vendor telemetry can't answer your question

Microsoft's numbers cover a broad population. They can't say whether your fleet looks like the half that patched quickly or the quarter that didn't. Your population has its own quirks: laptops that stay asleep in a bag, machines on slow networks, servers held back by change windows, and devices that never check in with a management tool.

## Measure your own patch gap

Start with two numbers: the share of hosts that had the October update at 14 days, and the share at 28 days. Fleet's agent reports installed software and OS version for each enrolled host, so you can build those counts per platform and per fleet, then compare them with Microsoft's figures.

Check the [Fleet documentation](https://fleetdm.com/docs) for the exact tables and fields available on each platform before you build the report. Devices that aren't enrolled won't appear in any count, so enrollment coverage is part of the result.

## Turn the list into follow-up

For each lagging host, record an owner and a date. Some devices will have a reason: a pending reboot, a deferred update, or a change freeze. Others will be devices nobody has looked at in weeks. Re-run the same report on the date you set and record what changed.

## The through-line

A patch release is a claim about what's fixed. Your inventory tells you where that claim is true. Teams that can measure their own gap don't need to borrow anyone else's telemetry.

## See it live

- **Get a demo** to see patch status across your own devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles software and vulnerability management:** [fleetdm.com/software-management](https://fleetdm.com/software-management)

<meta name="articleTitle" value="Microsoft's October update closed 1,100 CVEs. Which of your devices are still waiting?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-06">
<meta name="description" value="Microsoft says under half of enterprise devices had October's 1,100-CVE update after two weeks. Measure the patch gap on your own hosts.">
