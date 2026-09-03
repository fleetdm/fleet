# What 13 new FFmpeg vulnerabilities mean for tracking a media library most teams forget they run

*Ubuntu patched 13 vulnerabilities in FFmpeg, a media library quietly bundled inside countless internal tools, and several of the fixes only ship through Ubuntu Pro's Extended Security Maintenance. Here's how to find every host running a vulnerable build regardless.*

## Key takeaways

- **FFmpeg turns up in places nobody thinks to check.** It's the media backend behind countless internal tools and scripts that touch video, audio, or subtitle files, and the 13 CVEs Ubuntu patched span subtitle, audio, and video demuxers most teams never audit directly.
- **Two of the thirteen bugs go beyond a crash.** The VobSub subtitle demuxer flaw (CVE-2026-64830) and the TDSC video decoder flaw in AVI files (CVE-2026-65703) can both lead to code execution when a vulnerable build processes a malicious file, not just a denial of service.
- **The fix isn't automatic on any affected release.** Every one of Ubuntu's patched FFmpeg builds, from 16.04 LTS through 24.04 LTS, ships through Extended Security Maintenance, so a host without an active Ubuntu Pro subscription won't pull the fix through a routine apt upgrade.
- **You can find every host running FFmpeg without knowing where it's installed.** Fleet's software inventory reports installed package versions for FFmpeg the same way it does for anything else, so a dependency buried inside some internal tool still shows up.
- **Exposure depends on the Ubuntu release, not just the FFmpeg version.** Several of the 13 fixes apply only to specific releases, such as the ADX audio decoder bug that affects 22.04 and 24.04 LTS but not older releases, so the same package version can mean different exposure on different hosts.
- **A saved policy catches the next FFmpeg advisory too.** FFmpeg ships security notices often enough that a one-time sweep goes stale fast, and a Fleet policy keeps the answer current without anyone re-running the check by hand.

<a purpose="cta-button" href="https://fleetdm.com/security-and-control">See vulnerability management in Fleet</a>

Ubuntu shipped [USN-8716-1](https://ubuntu.com/security/notices/USN-8716-1) on September 3, 2026, patching 13 vulnerabilities in FFmpeg spanning its subtitle, audio, and video demuxers, muxers, and decoders. FFmpeg isn't the kind of software most teams inventory on purpose. It's the media engine quietly bundled inside transcoding scripts, internal media tools, CI pipelines that process video assets, and no shortage of third-party applications, which makes "do we run a vulnerable build" a harder question than it sounds.

Two of the 13 bugs raise the stakes beyond the usual denial-of-service patch batch: they can lead to arbitrary code execution if a vulnerable host processes the wrong file.

## Which of the 13 bugs actually matter most

Most of the batch is denial-of-service territory: crafted RTP/ASF streams, DTS audio in the S/PDIF muxer, ADX audio files, ffconcat handling in the TY demuxer, the vf_floodfill and vf_swaprect video filters, the Librist protocol handler, the VC2 HQ RTP packetizer, and DASH manifest processing (CVEs 2026-64833, -64834, -64835, -65704, -65705, -65706, -75142, -75143, -75144, and -75146) can crash a process that parses a malicious file, but don't go further on their own.

Two stand apart. CVE-2026-64830, in the VobSub subtitle demuxer, and CVE-2026-65703, in the TDSC video decoder's handling of AVI files, can lead to code execution rather than just a crash. A subtitle file is an easy thing to hand to someone or attach to a support ticket, which makes the VobSub bug worth prioritizing even though subtitle parsing sounds like the least security-critical corner of a media library. CVE-2026-75141, an HEVC parser flaw in hvcC NAL array handling, rounds out the code-execution-capable bugs in the batch.

## The patch isn't automatic on every affected release

This is the detail easy to miss. FFmpeg's fixed versions in USN-8716-1 carry an `+esm` suffix across the board, from `7:6.1.1-3ubuntu5+esm13` on 24.04 LTS down to `7:2.8.17-0ubuntu0.1+esm17` on 16.04 LTS, which means these fixes ship through Ubuntu Pro's Extended Security Maintenance rather than the standard archive. A host without an active Ubuntu Pro subscription attached won't receive these updates through a routine `apt upgrade`, no matter how current its other packages are. That gap matters more for FFmpeg than for a package most teams patch deliberately, because FFmpeg is so often a transitive dependency nobody remembers to check.

## Finding every FFmpeg install across your fleet

You don't need to know where FFmpeg is installed to find it. Fleet's software inventory reports installed package versions the same way for FFmpeg as it does for any other software on a host:

```sql
-- Debian/Ubuntu hosts
SELECT name, version FROM deb_packages WHERE name LIKE '%ffmpeg%';

-- RHEL/Fedora-family hosts
SELECT name, version FROM rpm_packages WHERE name LIKE '%ffmpeg%';
```

Match the returned version, and the host's Ubuntu release, against the fixed build for that release. Because several of the 13 CVEs only apply to specific releases, the same FFmpeg version can represent different real exposure depending on which Ubuntu release it's running on.

## Turning the sweep into a policy

A single query answers whether you're exposed today. A saved Fleet policy answers it every day, checking new and existing hosts against the correct fixed version for their release without anyone re-running the hunt when the next FFmpeg advisory lands, which given FFmpeg's patch cadence, won't be long. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as everything else, updating the version threshold for the next advisory is a reviewable pull request, not an undocumented console edit.

## A dependency this common deserves a standing answer

FFmpeg being everywhere is exactly why "we'll patch it when we notice" doesn't hold up. The next FFmpeg advisory will land on the same forgotten scripts and bundled tools as this one, and whether you already have an answer for "which hosts are exposed" is what determines how long that window stays open.

## See it live

- **Get a demo** to see software inventory and vulnerability matching against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already tracks across your hosts: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- Ubuntu, [USN-8716-1: FFmpeg vulnerabilities](https://ubuntu.com/security/notices/USN-8716-1).

---
*See which of your hosts are still running a vulnerable FFmpeg build. [Talk to Fleet](https://fleetdm.com/contact), or explore the [software catalog](https://fleetdm.com/software-catalog) Fleet already builds from your fleet.*

<meta name="articleTitle" value="What 13 new FFmpeg vulnerabilities mean for tracking a media library most teams forget they run">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-03">
<meta name="description" value="Ubuntu patched 13 FFmpeg bugs, some fixed only via Ubuntu Pro. See how to find every host running a vulnerable build with Fleet.">
