# Ubuntu's first .NET fix for CVE-2026-102870 didn't hold, so check for the second

*USN-8958-1 corrects USN-8942-1, which Ubuntu found to be incomplete. A host that took only the first update is still exposed, and "patched" in your records may not mean what you think.*

## Key takeaways

- **The first patch note closed nothing.** Ubuntu says the fix for CVE-2026-102870 in USN-8942-1 was incomplete. USN-8958-1, published October 29, finishes the job.
- **The bug is in the Blazor WebAssembly debug proxy.** It didn't properly escape certain values, which could allow arbitrary code execution. Only the dotnet10 and dotnet11 packages are affected.
- **The same notice carries other fixes.** It adds a Brotli decompression fix (CVE-2026-111497) and a local privilege escalation fix for .NET file ownership handling (CVE-2026-114148).
- **A version string is the evidence.** Each Ubuntu release has a specific fixed package version. A host either reports it or doesn't.
- **Re-check hosts you already marked done.** Anyone who closed the ticket after USN-8942-1 needs to look again.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See package versions across your hosts</a>

When a patch turns out to be incomplete, the original ticket is still closed in your tracker. That's the risk: a team marks CVE-2026-102870 done after the first Ubuntu notice, and the hosts keep running the version that didn't fully fix it.

Ubuntu's notice USN-8958-1 says exactly that. The fix released in USN-8942-1 was found to be incomplete, and this update addresses the remaining issue.

## What USN-8958-1 fixes

The notice covers the dotnet8, dotnet10, and dotnet11 packages and three CVEs.

CVE-2026-102870 is the Blazor WebAssembly debug proxy flaw. The proxy didn't properly escape certain values, which could potentially allow arbitrary code execution. Only dotnet10 and dotnet11 are affected.

CVE-2026-111497 is a Brotli decompression flaw. The library mishandled certain compressed streams, which could let a remote attacker cause denial of service or potentially execute arbitrary code.

CVE-2026-114148 is a local privilege escalation. .NET didn't properly handle file ownership in certain circumstances.

## The versions that count

Ubuntu lists a fixed package version per release. For dotnet10, the notice gives 10.0.14-0ubuntu1~24.04.1 on 24.04 LTS, 10.0.14-0ubuntu1~26.04.1 on 26.04 LTS, and 10.0.14-0ubuntu1~26.10.1 on 26.10. For dotnet8, it gives 8.0.33-0ubuntu1~24.04.1 on 24.04 LTS and 8.0.33-0ubuntu1~22.04.1 on 22.04 LTS. The SDK packages carry their own versions, so read the notice for the exact strings before you automate a comparison.

Because the first fix was incomplete, a host that reports an earlier dotnet10 build may look recently patched and still be exposed.

## Confirming the second patch landed

Fleet's agent reports installed package versions for each Linux host. A report against the `deb_packages` table lists what each host has:

```sql
SELECT name, version
FROM deb_packages
WHERE name LIKE 'dotnet%' OR name LIKE 'aspnetcore%';
```

Compare each result against the fixed version for that host's Ubuntu release. A host below it still needs the update. A host with no .NET packages isn't in scope. Package names vary by what's installed, so check the output against your own hosts before you build a policy from it.

## Closing

The lesson of USN-8958-1 isn't about .NET. A patch note says a fix shipped, not that it worked, and not that your hosts have the version that works. Reading the installed version from each host is the only check that survives a corrected advisory.

## See it live

- **Get a demo.** See package versions across your Linux hosts: [fleetdm.com/contact](https://fleetdm.com/contact)

## Sources

- Ubuntu Security Notices, [USN-8958-1: .NET vulnerabilities](https://ubuntu.com/security/notices/USN-8958-1).

<meta name="articleTitle" value="Ubuntu's first .NET fix for CVE-2026-102870 didn't hold, so check for the second">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-30">
<meta name="description" value="Ubuntu's USN-8958-1 corrects an incomplete .NET fix for CVE-2026-102870. Confirm which Linux hosts picked up the corrected package version.">
