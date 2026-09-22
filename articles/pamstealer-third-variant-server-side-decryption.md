<meta name="category" value="industry news">
<meta name="articleTitle" value="PamStealer's third variant now needs the attacker's permission to decrypt its own payload">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-09-22">
<meta name="description" value="PamStealer's newest variant won't decrypt without a live server handshake. See how to find the fake Wavel wallet installer on your Macs with Fleet.">

# PamStealer's third variant now needs the attacker's permission to decrypt its own payload

*Jamf Threat Labs found a new PamStealer variant that generates a fresh encryption key with its command-and-control server on every run, so the malware can't be unlocked for analysis without the attacker's live cooperation. Here's how to find the fake installer it rides in on before it gets that far.*

## Key takeaways

- **You don't need to break PamStealer's encryption to catch it. You need to find the installer before someone runs it.** Fleet's agent already inventories installed applications and can check for the DMG and dropper files on every Mac, so a team can act on the delivery mechanism instead of waiting to unpack an intentionally locked payload.
- **This variant is distributed as Wavel, a fake cryptocurrency wallet.** Earlier PamStealer variants posed as a clipboard manager called Maccy; the lure changes each time, so name-based blocklists lag the campaign by design.
- **The payload now can't be decrypted without a live round trip to the attacker's server.** Each run generates a fresh X25519 keypair and exchanges it with command-and-control infrastructure before the malware can even unwrap itself, unlike earlier variants that carried their decryption key in the code.
- **That design defeats static analysis on purpose.** A security team that captures the dropper offline gets an encrypted blob, not a payload, because the decryption key never ships with the file.
- **The delivery format is a known, queryable pattern.** All three PamStealer variants use the same compiled JXA outer format, a signature Fleet can check for by file characteristics even as the lure and the C2 infrastructure keep changing.
- **Turn the file check into a standing policy.** Save it once and every Mac that downloads the current lure, or the next one, gets flagged automatically instead of waiting for a compromised credential to surface the infection after the fact.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Jamf Threat Labs identified a third variant of PamStealer, a macOS infostealer family, distributed as Wavel, a fake cryptocurrency wallet app served from a lookalike site. What sets this variant apart isn't what it steals, it's how hard it is to analyze after the fact. Earlier PamStealer samples embedded their decryption key directly in the malicious script, so a researcher who captured the file could read the payload offline. This one doesn't ship with a key at all. It fetches a purpose-built decryption tool and completes a key exchange with the attacker's server before it can unwrap itself, and that exchange uses fresh, single-use keys every time.

That's a deliberate anti-analysis move, not an incidental one. A team that captures the dropper without the C2 infrastructure cooperating gets an encrypted blob and nothing else. The practical answer isn't to out-engineer that design. It's to catch the installer before it runs, which is a question about what's on the machine, not about breaking encryption designed specifically to resist that.

## How this variant locks its own payload

The earlier PamStealer variant, distributed as a fake Maccy clipboard manager, carried its AES decryption key inside the compiled JXA source. Anyone who obtained the dropper could extract the key and read the payload without ever running it.

This variant removes that shortcut. On execution, it downloads a separate binary, `pkgunpack`, that generates an ephemeral X25519 keypair and posts the public key and a nonce to the attacker's server. The server responds with an encrypted data encryption key, and `pkgunpack` derives the unwrapping key through an X25519 exchange with the server's hardcoded public key before decrypting the actual payload with AES-256-GCM. Every run generates new ephemeral keys, so a decrypted sample from one infection doesn't help unlock another, and the payload simply doesn't exist in readable form unless the attacker's infrastructure is live and responds.

## What stays constant even as the lure changes

Jamf has now tracked three PamStealer variants: a fake Maccy clipboard manager, this fake Wavel wallet, and one in between. The lure keeps changing, and it will keep changing, because a fake app name is the cheapest part of the campaign to swap out. Chasing the current lure name is chasing a moving target.

What hasn't changed across all three is the delivery format: a compiled JXA dropper, a binary that opens with the magic bytes `JsOsaDAS1.001.00` followed by a binary property list containing UTF-16BE-encoded JXA source. That structural signature is the more durable thing to look for, because it doesn't change when the campaign swaps its fake app's name and icon next month.

## Finding it before the payload matters

Fleet's agent already inventories installed applications and files across every Mac it manages, so the check here doesn't depend on ever seeing what's inside the encrypted payload. It depends on whether the fake installer or its dropper landed on a machine in the first place.

Start with the application layer. If the current campaign's fake Wavel app or its DMG made it onto a host, Fleet's software inventory will show it the same way it shows any other installed application, which means a team can search for the known bad app name across the fleet in seconds. Because that name will change in the next campaign, pair the name-based check with a Fleet run-script that inspects candidate dropper files for the JXA binary's magic-byte signature, so detection survives the next rebrand instead of expiring with it.

## Turning the check into a standing policy

A one-time sweep tells a team about today's exposure. Saving the check as a Fleet policy, matched against both the current known-bad app names and the structural JXA signature, means every Mac that pulls down this lure, or the next one built the same way, gets flagged as it happens. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as the rest of a team's configuration, updating the list of known lures as Jamf and others publish new indicators is a reviewable pull request, not a manual edit somebody has to remember to make.

## The lock is the tell

PamStealer's operators built a payload that resists analysis by design, and that's worth taking seriously as a sign of how deliberate this campaign is. But it doesn't change where the actual opportunity to catch it sits. The malware only matters once someone has already run a fake installer, and that installer, and the dropper format underneath it, are both things Fleet can already see on a Mac before that happens.

## See it live

- **Get a demo** to see software inventory and file-based detections against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already tracks across your Macs: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- Jamf Threat Labs, [New PamStealer variant targets macOS via fake crypto wallet](https://www.jamf.com/blog/pamstealer-wavel-macos-infostealer/).
- Jamf Threat Labs, [PamStealer: macOS Malware Posing as Clipboard Manager App](https://www.jamf.com/blog/pamstealer-macos-infostealer-applescript-rust/).
