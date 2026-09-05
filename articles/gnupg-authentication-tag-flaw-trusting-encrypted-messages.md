# What Ubuntu's GnuPG authentication tag flaw means for trusting encrypted messages

*A GnuPG bug let attackers craft AES-GCM encrypted messages that gpgsm would treat as authentic without a real authentication tag behind them. Here's how to confirm the patched build reached every host.*

## Key takeaways

- **Authenticated encryption stops meaning anything if the "authenticated" part is skippable.** AES-GCM's authentication tag exists to prove a ciphertext wasn't tampered with in transit; gpgsm's flawed validation let attackers supply a tag far shorter than the algorithm requires and still pass verification.
- **This is an S/MIME bug, not a PGP one.** CVE-2026-57062 lives in gpgsm, GnuPG's CMS and S/MIME component, so it's about handling CMS-wrapped and S/MIME-encrypted content specifically, not classic OpenPGP-format encryption.
- **The fix is release-specific, so "patched" isn't one number.** Ubuntu shipped gpgsm 2.4.8-4ubuntu3.1 for 26.04 LTS and 2.4.4-2ubuntu17.6 for 24.04 LTS, two different target versions depending on which release a host runs.
- **You can confirm the exact gpgsm build on every host without asking around.** Fleet's software inventory reports the installed gpgsm version the same way it reports any other package, so confirming the patch landed is a query, not an assumption.
- **A version check only protects messages processed after the patch.** Anything gpgsm accepted as authentic before the fix landed can't be retroactively re-verified, so a host that was exposed during that window needs more than a version bump.
- **Saved as a policy, this stops being a one-time question.** A Fleet policy comparing installed gpgsm against the patched build for each release answers "are we covered" continuously instead of just the day someone remembered to check.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Ubuntu shipped [USN-8720-1](https://ubuntu.com/security/notices/USN-8720-1) on September 3, 2026, fixing a flaw in how gpgsm, GnuPG's S/MIME (CMS) component, validates authentication tags on messages encrypted with AES-GCM. The bug, tracked as CVE-2026-57062, meant gpgsm didn't properly enforce that an AES-GCM authentication tag was long enough to actually prove anything, which let a specially crafted CMS message slip past the integrity check its encryption was supposed to guarantee.

That's a quieter kind of failure than a typical crypto bug. Nothing crashes and no error appears. A message that should have been rejected as tampered gets treated as authentic instead.

## Why a short authentication tag defeats the point of AES-GCM

AES-GCM is an authenticated encryption mode: alongside the encrypted content, it produces a tag that proves the ciphertext wasn't modified after encryption. If a decryptor doesn't enforce a real tag length, an attacker can craft ciphertext with an authentication field too short to carry any cryptographic weight and have it accepted anyway. Ubuntu's fix tightens that check, enforcing a valid tag length and rejecting non-AEAD ciphers during AuthEnvelopedData processing, so a message either carries genuine proof of integrity or gets rejected outright.

## This affects S/MIME handling, not classic PGP

It's worth being precise about what's exposed. The flaw lives in gpgsm, the part of GnuPG that handles CMS and S/MIME, not the OpenPGP-format encryption most people associate with GnuPG. If your workflows use gpgsm to verify signed or encrypted S/MIME mail or CMS-wrapped payloads, this bug is relevant. If they don't touch S/MIME at all, this specific flaw isn't your exposure, though keeping gpgsm current is still worth doing.

## Confirming the patched build across your fleet

Ubuntu's fix isn't one version number, it's two: gpgsm 2.4.8-4ubuntu3.1 for Ubuntu 26.04 LTS and 2.4.4-2ubuntu17.6 for Ubuntu 24.04 LTS. Fleet's software inventory reports the installed gpgsm version on every Linux host the same way it reports any other package, so checking whether the fix landed is a query, not a guess:

```sql
SELECT name, version FROM deb_packages WHERE name = 'gpgsm';
```

Compare the returned version against the fixed build for that host's Ubuntu release. Anything older is still running the flawed tag check.

## Turning the check into an ongoing policy

A one-time query answers "are we patched today." A saved Fleet policy answers it every day going forward, checking new and existing hosts against the correct fixed version for their release without anyone re-running the query when the next GnuPG advisory lands. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as the rest of your configuration, updating the target version is a reviewable pull request, not a console edit somebody has to remember to make.

## The patch protects what comes next, not what already happened

Confirming gpgsm's version tells you the flawed check is closed going forward. It doesn't tell you whether a tampered message was accepted as authentic before the patch landed. If a host processed CMS or S/MIME messages during the exposure window, treat anything decrypted or verified during that window as unconfirmed until you can re-establish its integrity some other way.

## See it live

- **Get a demo** to see software inventory and vulnerability matching against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already tracks across your hosts: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- Ubuntu, [USN-8720-1: GnuPG vulnerability](https://ubuntu.com/security/notices/USN-8720-1).

---
*See which of your hosts are still running the vulnerable build. [Talk to Fleet](https://fleetdm.com/contact), or explore the [software catalog](https://fleetdm.com/software-catalog) Fleet already builds from your fleet.*

<meta name="articleTitle" value="What Ubuntu's GnuPG authentication tag flaw means for trusting encrypted messages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-03">
<meta name="description" value="A GnuPG bug let tampered S/MIME messages pass integrity checks. See how to confirm the patched gpgsm build reached every host.">
