# curl and git on your Macs trusted compromised eMudhra certificates until macOS 27.2.1

*Apple blocked the eMudhra intermediates for Safari and Mail in October. The command-line tools on the same Mac needed a second update, and a build runner counts as a Mac too.*

## Key takeaways

- **Two trust stores, two fix dates.** Apple's TLS stack blocked the compromised eMudhra intermediates on October 19. Tools that read `/etc/ssl/cert.pem`, including curl and git, kept accepting them until a later update.
- **Three macOS versions carry the command-line fix.** Apple lists macOS 27.2.1, macOS 26.7.3, and macOS 15.8.3. A Mac on any older build is still exposed through its Terminal tools.
- **Build machines are the quiet risk.** CI runners and developer laptops run curl and git all day, often pulling code and dependencies over TLS. They are managed endpoints, even when nobody thinks of them as laptops.
- **"Updated" is a number you can count.** The OS version is reported per host, so the question becomes how many Macs are below the fixed version, not whether the update was announced.
- **The certificate file is the direct check.** A policy that reads `/etc/ssl/cert.pem` confirms the exposure is gone on each Mac, including Macs where someone applied a manual workaround.

<a purpose="cta-button" href="https://fleetdm.com/guides/find-and-fix-emudhra-intermediate-ca-exposure">Find and fix eMudhra exposure</a>

On October 20, 2026, Mozilla disclosed that eMudhra generated the private keys of eight intermediate certificate authorities incorrectly. Anyone who recovers a key can issue a certificate that a vulnerable TLS client accepts for any domain. Apple's October fix covered Safari, Mail, and apps built on Apple's frameworks.

It did not cover the command-line tools. Apple's advisory says tools included with macOS or the Xcode Command Line Tools, such as curl and git, check certificates against `/etc/ssl/cert.pem` and keep accepting the blocked certificates until they get an update. That update is macOS 27.2.1. Which of your Macs have it?

## Why the October fix didn't reach Terminal

Safari and apps built on Apple's frameworks ask the operating system to verify a certificate, and the operating system knows the intermediates are blocked. curl, git, python3, ruby, perl, and openssl take a different route. They read a static bundle of trusted roots and don't check revocation, so a root in that file is a root they trust.

Two roots in that bundle matter here: emSign ECC Root CA - G3 and AAA Certificate Services. The compromised intermediates chain to them. Until the bundle or the tool changes, a certificate issued with a recovered key looks valid to these tools.

## What Apple's update changes

Apple's support page says that starting in macOS 27.2.1, the command-line tools stop accepting the blocked certificates. The page adds that macOS 26.7.3 and macOS 15.8.3 include the same protection. Apple's page doesn't give a release date for the updates themselves.

For Macs that can't update yet, Apple describes removing the affected root certificates from `/etc/ssl/cert.pem` as a workaround. Reports from earlier in the month say a system update can restore that file, which is a reason to keep checking after any manual change.

## Why your pipeline counts as an endpoint

A developer laptop runs git against your source host all day. A CI runner pulls dependencies, downloads tooling with curl, and often holds credentials for the systems it deploys to. A certificate the tool should have rejected, presented in the right place, is a way to hand that machine something it shouldn't trust.

Teams tend to track laptops closely and runners loosely. A Mac mini in a rack that builds your iOS app is on the same macOS release train as everything else, and it's the machine most likely to sit on an old build because nobody logs into it.

## Counting which Macs picked up the fix

Fleet reports each host's macOS version. A policy can turn Apple's list into a pass or fail for every Mac. This query passes on macOS 27.2.1 or later, 26.7.3 or later, and 15.8.3 or later within their own release lines:

```sql
SELECT 1 FROM os_version
WHERE platform = 'darwin' AND (
  major > 27
  OR (major = 27 AND (minor > 2 OR (minor = 2 AND patch >= 1)))
  OR (major = 26 AND (minor > 7 OR (minor = 7 AND patch >= 3)))
  OR (major = 15 AND (minor > 8 OR (minor = 8 AND patch >= 3)))
);
```

The failing list is your work queue: Macs that need the update, or that have to stay on a manual fix until they can take it. The count drops as updates land.

## Checking the file itself

Version numbers tell you what Apple shipped, not what each Mac is doing. Fleet's guide for the eMudhra compromise includes a policy that reads `/etc/ssl/cert.pem` and passes only when neither compromised root is present. It also covers fleetd's own CA bundle and includes a script that removes the roots from Macs that can't update yet.

Use both checks. The version policy says whether Apple's fix is installed. The file policy says whether the exposure is closed, including on a Mac where someone already ran the workaround.

## The takeaway for build machines

This week's update closes the gap for Macs that install it. The habit that outlasts the incident is treating runners, build hosts, and lab machines as part of the same inventory as laptops, with the same question asked of each: which build is it on, and is that the fixed one?

## See it live

- **Find and fix eMudhra exposure.** Follow the guide with ready-to-use policies and scripts: [eMudhra intermediate CA guide](https://fleetdm.com/guides/find-and-fix-emudhra-intermediate-ca-exposure)
- **Get a demo.** See macOS versions and certificate policies across your hosts: [fleetdm.com/contact](https://fleetdm.com/contact)

## Sources

- Apple Support, [About the compromised eMudhra intermediate CAs](https://support.apple.com/en-la/149555).
- Fleet, [Find and fix exposure to the eMudhra intermediate CA compromise](https://fleetdm.com/guides/find-and-fix-emudhra-intermediate-ca-exposure).

<meta name="articleTitle" value="curl and git on your Macs trusted compromised eMudhra certificates until macOS 27.2.1">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-10">
<meta name="description" value="Apple's eMudhra fix reached curl and git only in macOS 27.2.1. Count which Macs, including CI runners, are below the fixed version.">
