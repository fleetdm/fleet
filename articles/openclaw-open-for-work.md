# OpenClaw: Open for work?

### Links to article series:

- Part 1: OpenClaw: Open for work?
- Part 2: [Detecting AI agents like OpenClaw with automated tooling](https://fleetdm.com/articles/detecting-ai-agents-like-openclaw-with-automated-tooling)
- Part 3: [Mitigation assets and detection patterns for AI agents like OpenClaw](https://fleetdm.com/guides/mitigation-assets-and-detection-patterns-for-ai-agents-like-openclaw)

The amount of recent chatter about [OpenClaw](https://openclaw.im/) seems to be highlighting a [cultural inflection point](https://www.jakequist.com/thoughts/openclaw-is-what-apple-intelligence-should-have-been). Like the point at which everyone switched off AOL and on to the regular old internet. The point at which everyone suddenly had an iPod. The point at which everyone had a Gmail account. 

This may be the point at which useful AI technology has finally become attainable for people who would not consider themselves technology-savvy. Powerful, connected, practical. Easy to set up, use and understand. 

## How OpenClaw works

OpenClaw integrates out-of-the-box with [WhatsApp](https://www.whatsapp.com/), [Telegram](https://telegram.org/), [Discord](https://discord.com/), and Apple's [iMessage](https://support.apple.com/guide/iphone/about-imessage-iph4e9799206/ios). Users talk to their OpenClaw agent via chat. That's not new. What is new is how easy it is to integrate OpenClaw with systems that have never been easy to connect before using only chat (or pictures or audio) as input for AI agents and skills.

Linking capabilities together with systems like Apple Shortcuts and similar tools has been possible for years, but users had to build the connections and rules themselves. That is no longer necessary. 

Other OpenClaw benefits:

- OpenClaw is open-source.
- OpenClaw can run on a computer that the end user controls.
- There is no need to access a corporate web app portal or third-party corporate app to use it.
- The documentation includes hundreds of pre-built, easy-to-load skills that integrate the agent into 1000's of possible workflows

## What are the risks of OpenClaw? 

Running OpenClaw in your own home on a dedicated computer does provide a basic security advantage. But easing the barrier to entry for technology always presents risks. This is true for individuals running OpenClaw at home and especially true for anyone considering using OpenClaw in the enterprise, where organizations try to limit liability, comply with regulations and laws, and protect investments in assets and people.  

Simply put, OpenClaw is intended to run as root on the computer where it's installed. It works best with full access to [Transparency, Consent and Control (TCC)](https://support.apple.com/guide/security/controlling-app-access-to-files-secddd1d86a6/web) user privileges on macOS, meaning it can access any app using user space data, biometrics, the microphone, or the camera. It can use skills and connected AI agents to navigate almost any installed third-party app, web app, or page on the internet. It will even try to find devices on your network via [mDNS](https://multicastdns.org/) discovery.

It is this extensible integration capability and the authority OpenClaw users grant to the agent (access to authenticate "as you" via two-factor authentication (2FA), access to bank accounts, medical records, contacts, calendars, email, etc.) that gives the system its power. It can basically do anything an end user can by stringing together multiple apps & human intelligence. It just doesn't need a human to be involved. This is thrilling from a technology perspective (which is why everyone is talking about it) and daunting from a security perspective.

### Prompt injection

The biggest potential risk given how OpenClaw works is [prompt injection](https://www.ibm.com/think/topics/prompt-injection). 

Shared computing systems from the time of Unix in the 1970s until now have always had built-in protections and layers of security. Many of the original ideas created to keep operating systems secure (e.g., the [sudo](https://www.sudo.ws/) command) still work well.

But, OpenClaw's capabilities run with an unprecedented level of autonomy. In the user space, with root privilege (if given). No human interaction, know-how, skill, or experience is required - just a text message.

The real danger of prompt injection is that if someone other than the intended user can send text messages to an OpenClaw agent, even with its "intelligence", the system has virtually no ability to discern an order coming from you versus an order from someone pretending to be "you". 

Prompt injections can also be indirect, meaning an autonomous agent may encounter a hidden command or script on a website it was instructed to navigate, or in a malicious email attachment, without recognizing a problem.

## Can device management help?

Our [next article](https://fleetdm.com/articles/detecting-ai-agents-like-openclaw-with-automated-tooling) (part 2 of this series) will explore how to understand this new threat and what IT leaders should know about protecting their environments when autonomous AI agents are in use.

<meta name="articleTitle" value="OpenClaw: Open for work?">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-04">
<meta name="description" value="Part 1 of 3 - OpenClaw: An overview, how it works, and the security risks of autonomous AI agents on user devices.">
