# Why work laptops don't work on plane wifi

You're on a plane. You open your work laptop, connect to the inflight wifi, and… nothing works right.

The captive portal won't load. Your browser throws certificate errors. Slack won't connect. You can't reach your email. You try a few things, give up, and close the lid.

This happens constantly. And it's not because plane wifi is bad (although it often is). It's because the security tools on your work laptop weren't designed for this environment.

## What's actually going wrong

Most IT teams configure work laptops with layers of security: DNS filtering, always-on VPNs, endpoint protection agents, firewall rules, and certificate-based authentication. These tools assume a "normal" network — your home wifi, a corporate office, or a coffee shop with a standard captive portal.

Plane wifi is anything but normal.

Here's what typically breaks:

- **Captive portals get blocked.** Many DNS filters and VPNs intercept traffic before you can even reach the airline's login page. The portal never loads, so you can't authenticate to the network.
- **VPN tunnels fail silently.** Always-on VPNs try to connect immediately, but plane wifi often uses aggressive NAT, bandwidth throttling, or blocks VPN protocols entirely. The VPN can't connect, and because it's configured to block all traffic outside the tunnel, nothing works.
- **DNS filtering interferes.** Corporate DNS resolvers may not be reachable from the plane's network, causing every lookup to fail or time out.
- **Certificate errors appear.** Some inflight networks use SSL interception or redirect HTTPS traffic in ways that conflict with your laptop's certificate pinning or trust store.

The result: your laptop is online, but functionally useless.

## Why employees can't fix it

Here's the UX problem. Most employees don't have admin access. They can't disable the VPN, change DNS settings, or adjust firewall rules. Even if they could, they wouldn't know which tool is causing the issue.

Error messages are unhelpful. "Unable to connect" doesn't tell you whether the problem is the VPN, the DNS filter, the captive portal, or the plane's network itself. There's no diagnostic tool that says, "Your VPN is blocking the captive portal — here's what to do."

So employees sit there, frustrated, unable to work for a 4-hour flight. That's lost productivity. It's also a bad experience that erodes trust in IT.

This came up recently in a [LinkedIn discussion](https://www.linkedin.com/feed/update/urn:li:activity:7427732997539274753) where IT professionals and frequent travelers shared their frustrations. The thread made one thing clear: this is a widespread, unsolved problem.

## What if work laptops just worked on plane wifi?

Here's the thesis worth exploring: maybe IT teams should make employee laptops work on plane wifi by default.

Not "work perfectly with full security." Just… work. Let people open a browser, reach the captive portal, authenticate, and get basic connectivity. Email, Slack, docs. The essentials.

What would that look like in practice?

**Smarter VPN configurations.** Split-tunnel VPNs that route corporate traffic through the tunnel but let general internet traffic flow directly. Or VPN clients that detect captive portals and pause the tunnel until authentication completes.

**Captive portal detection.** macOS and Windows both have built-in captive portal detection, but security tools often interfere with it. IT teams could configure their tools to allow captive portal flows before enforcing the full security stack.

**Graceful degradation.** Instead of "all or nothing" security, policies could adapt to the network environment. On a restricted network like plane wifi, the laptop could fall back to a reduced-security mode — still running endpoint protection, still encrypting traffic where possible, but not blocking everything because the VPN can't connect.

**Better diagnostics.** Give employees a simple status indicator: "You're on a restricted network. Some features may be limited." That's better than a wall of cryptic errors.

## The tradeoff is real, but manageable

Yes, relaxing security policies on certain networks introduces risk. But the current situation has its own risks: employees tethering to personal phones to bypass restrictions, or using personal laptops without any security tools at all.

A managed, intentional approach to restricted networks is better than pretending the problem doesn't exist.

IT teams already make tradeoffs like this. Guest networks, BYOD policies, and travel configurations all involve balancing security with usability. Plane wifi is just another environment that deserves a thoughtful policy.

## Where to start

If you manage a fleet of devices, consider auditing what happens when one of your laptops connects to plane wifi. Try it yourself. See what breaks. Then ask: is this the experience we want our employees to have?

The answer is probably no. And the fix might be simpler than you think.

<meta name="articleTitle" value="Why work laptops don't work on plane wifi">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-13">
<meta name="description" value="Work laptops often break on plane wifi due to VPNs, DNS filters, and captive portal conflicts. Maybe IT teams should make them work by default.">
