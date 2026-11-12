# Monitor DNS traffic on macOS

If you're deploying an EDR, content filter, or any security tool that needs visibility into DNS queries on your Mac fleet, you'll hit a macOS-specific challenge: all DNS resolution flows through a single system daemon, and Apple's privacy protections can hide queries from your tools entirely.

This article explains how DNS works under the hood on macOS, what APIs are available for monitoring it, and what you need to configure via MDM to make sure your security tooling actually sees the traffic.

## Why DNS monitoring matters for your fleet

DNS is one of the most valuable telemetry sources for security teams. Malware phones home via DNS. Data exfiltration tunnels through DNS. Command-and-control channels hide in DNS. If your endpoint security tool can't see DNS queries, it's flying blind.

On macOS, getting this visibility requires understanding how the OS handles DNS differently from Linux or Windows, and deploying the right MDM configuration to prevent encrypted DNS from creating blind spots.

## How macOS handles DNS

macOS routes all DNS resolution through `mDNSResponder`, a system daemon. When any application calls `getaddrinfo()` or similar APIs, the request goes to `mDNSResponder`, which performs the actual DNS lookup on the application's behalf. The DNS traffic on the network originates from `mDNSResponder`, not from the application that initiated the lookup.

This matters because traditional packet capture on macOS won't tell you which application made a DNS request. You need Apple's APIs for per-process attribution.

## Network connection monitoring with NEFilterDataProvider

Apple's `NEFilterDataProvider` network extension captures TCP/UDP socket flows with per-process attribution. Each `NEFilterSocketFlow` includes a `remoteHostname` property populated from the connection's destination when available. For WebKit-based apps like Safari, this reliably provides the hostname. For other apps (Chrome, Brave, Edge, etc.), you may only get an IP address, since the hostname availability depends on where the filter taps into the networking stack.

For many EDR use cases, this hostname attribution is sufficient for WebKit-based traffic. But it doesn't cover all browsers, plain HTTP, non-web protocols, or cases where you need to log the DNS query itself.

## DNS query monitoring with NEDNSProxyProvider

For full DNS query capture, Apple provides `NEDNSProxyProvider`. Once activated, `mDNSResponder` routes DNS queries through the proxy provider instead of sending them directly to the upstream resolver. This gives your security tool:

- Per-process attribution via the source application's audit token
- Visibility into all plain-text DNS queries (UDP/53, mDNS)
- The query name, type, and originating process for each lookup

The DNS proxy replaces the system resolver in the chain, so it must forward queries to an upstream resolver after capturing them.

If you're building or evaluating a tool that uses this API, [Objective-See's DNSMonitor](https://github.com/objective-see/DNSMonitor) is a good reference implementation.

## The encrypted DNS blind spot

DNS over HTTPS (DoH) and DNS over TLS (DoT) create a visibility gap that affects any macOS security tool relying on `NEDNSProxyProvider` for DNS monitoring.

### What happens

The `mDNSResponder` daemon discovers encrypted DNS support by querying for `_dns.resolver.arpa` records. If the upstream DNS server responds with DoH or DoT configuration, `mDNSResponder` upgrades to encrypted DNS automatically and bypasses `NEDNSProxyProvider` entirely. Apple confirmed this is the intended behavior ([Feedback FB11963304](https://developer.apple.com/forums/thread/729619)). The consequences:

- DoH traffic (port 443) is indistinguishable from normal HTTPS
- DoT traffic (port 853) is detectable by port, but the content is encrypted
- The DNS proxy provider never sees these queries

Your EDR may report zero DNS activity from a device, not because the device is idle, but because all its DNS is going over an encrypted channel your tool can't see.

### Mitigations

There is no MDM payload to simply "turn off encrypted DNS" on macOS. Apple's [`com.apple.dnsSettings.managed`](https://support.apple.com/guide/deployment/dns-settings-payload-settings-dep86469ba99/web) payload only supports two protocol values, `HTTPS` (DoH) and `TLS` (DoT), and is designed to *enable* encrypted DNS, not disable it.

Instead, you have several options depending on your environment:

**1. Point devices to DNS servers that don't support DoH/DoT**

The simplest approach. If the upstream resolver doesn't respond to `_dns.resolver.arpa` queries with DoH/DoT configuration, `mDNSResponder` won't upgrade to encrypted DNS. Configure your managed devices to use internal DNS servers via an MDM Wi-Fi or network profile, or through DHCP settings. This keeps all DNS in cleartext where `NEDNSProxyProvider` can see it.

**2. Block DoH/DoT discovery in your DNS proxy**

If you're building or deploying a tool that uses `NEDNSProxyProvider`, Apple's recommended workaround is to respond to `_dns.resolver.arpa` queries with NXDOMAIN. This prevents `mDNSResponder` from discovering that the upstream resolver supports encrypted DNS, keeping queries flowing through the proxy.

**3. Force DNS through your corporate resolver over DoH/DoT**

Use the `com.apple.dnsSettings.managed` payload to point all DNS to your own encrypted DNS server. This doesn't give `NEDNSProxyProvider` visibility on-device, but you get all queries server-side in your DNS logs. This is the right approach if your priority is centralized logging over per-device endpoint telemetry.

**4. Block known public DoH/DoT providers**

Firewall rules blocking outbound connections to known DoH resolver IPs (e.g., 8.8.8.8:443, 1.1.1.1:443) and port 853 (DoT) catch applications that hardcode DoH providers like Google or Cloudflare.

**5. SNI inspection**

Monitor TLS Client Hello messages for connections to hostnames like `dns.google` or `cloudflare-dns.com` on port 443, which indicate DoH usage even though the query content is encrypted. Note that Encrypted Client Hello ([ECH, RFC 9849](https://www.rfc-editor.org/rfc/rfc9849.html)) is increasingly adopted by major CDNs and can encrypt the SNI field, reducing the effectiveness of this approach.

**6. Network-level TLS inspection**

A TLS-intercepting proxy with a trusted root CA deployed to endpoints can decrypt and inspect DoH traffic. This is the only way to see actual query content when encrypted DNS is in use. The root CA and the DNS settings profile from mitigation #3 can both be deployed via MDM. If your organization uses Fleet, see the [custom OS settings guide](https://fleetdm.com/guides/custom-os-settings).

## Audit DNS configuration across your fleet with osquery

If you use Fleet or `osquery`, you can audit DNS resolver configurations across your devices. This is useful for verifying that your mitigations are actually in place.

### Check configured DNS resolvers

The `dns_resolvers` table shows which DNS servers each device is using:

```sql
SELECT * FROM dns_resolvers WHERE type = 'nameserver';
```

If you see public resolvers like `1.1.1.1` or `8.8.8.8` that support DoH/DoT, those devices may have an encrypted DNS blind spot. You can turn this into a Fleet policy that flags devices not pointing to your internal DNS servers.

### Detect active DoT connections

DNS over TLS uses port 853. You can check for active connections on that port:

```sql
SELECT p.name, p.path, pos.remote_address, pos.remote_port
FROM process_open_sockets pos
JOIN processes p ON p.pid = pos.pid
WHERE pos.remote_port = 853;
```

### Check for MDM profiles with DNS settings

If your organization deploys DNS configuration via MDM, you can verify the profile is installed:

```sql
SELECT * FROM managed_policies WHERE domain = 'com.apple.dnsSettings.managed';
```

### The visibility gap

`osquery` can tell you *which* DNS servers are configured, but it can't tell you *whether* `mDNSResponder` is using cleartext or encrypted DNS for its queries. There's no table that exposes the active DNS protocol. This is why preventing the upgrade in the first place, by pointing devices to resolvers that don't support DoH/DoT, is more reliable than trying to detect it after the fact.

## Summary

Getting DNS visibility on macOS comes down to three things: understanding that `mDNSResponder` centralizes all DNS, using the right network extension APIs for per-process attribution, and ensuring encrypted DNS doesn't create blind spots for your tools. There's no single MDM toggle to disable encrypted DNS. The most practical approach for most organizations is pointing managed devices to internal DNS servers that don't support DoH/DoT, so `mDNSResponder` never upgrades in the first place.

<meta name="articleTitle" value="Monitor DNS traffic on macOS">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-08">
<meta name="description" value="How DNS works on macOS, what APIs security tools use to monitor it, and how to prevent encrypted DNS from creating blind spots for your EDR.">
