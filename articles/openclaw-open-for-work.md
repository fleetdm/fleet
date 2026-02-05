# OpenClaw: Open for work?

Unless you actually live at the bottom of the ocean, it's been impossible to miss the tidal wave of news crashing around OpenClaw. (I had to start with puns. My own, by the way, not AI\!)

- Mac minis selling out-of-stock (I don't know if this is actually true...)  
- Scary "Black Mirror" stories about sentience...  
- Novel ways of making restaurant reservations…

## The OpenClaw movement

The amount of writing about OpenClaw seems to be highlighting a cultural inflection point. I can recall similar moments: the point at which everyone seemed to be switching off AOL and onto the regular old internet. The point at which everyone had an iPod. The point at which everyone suddenly had a Gmail account. 

We may indeed be witnessing the point where useful AI technology has finally become attainable for normal people who would not consider themselves technology savvy. Powerful, connected, practical. Easy to set up, use and understand. 

## How OpenClaw works

OpenClaw integrates out-of-the-box with WhatsApp, Telegram, Discord, and iMessage. You talk to your OpenClaw agent via chat. That's not new. What is new is how easy it is to integrate OpenClaw with systems that have never been easy to connect before using only chat (or pictures or audio) as input for AI agents and skills.

Imagine asking your OpenClaw agent via text to look at pictures on AirBNB listings for a travel destination to avoid rooms with "pull-out beds" or "shag carpet". 

Imagine getting detailed information about the make and model of a shoe, where you can buy it and then monitor prices at competing vendors for the rest of eternity simply by sending OpenClaw a text message with a picture. 

What about logging into your dentist's reservation portal for you just by asking? 

These types of connections have been possible before, but you had to build the connections. Now, you don't. The best feature of OpenClaw? It runs on a computer you own, not on a corporate web app portal or 3rd party corporate app. 

## What are the risks of OpenClaw? 

Running OpenClaw in your own home on a dedicated computer does have security advantages. But, easing the barrier to entry for any technology always presents risks. This is true for individuals running OpenClaw at home and especially true for anyone considering using OpenClaw in the enterprise where organizations try to limit liability, comply with regulations and laws, and protect investments in assets and people.  

Simply put, OpenClaw is intended to run as root on the computer where it's installed. It works best with full access to TCC user privileges on macOS, meaning it can access any app using user space data, biometrics, the microphone or the camera. It can use skills and AI agents to navigate almost any installed 3rd party app or web app on the internet. 

It is this extensible integration capability and the authority OpenClaw users grant to the agent (access to authenticate "as you" via two-factor authentication (2FA), access to bank accounts, medical records, contacts, etc.) that gives the system its power: it can basically do anything you can do with multiple apps & human intelligence. It just doesn't need a human to be involved. This level of autonomy is thrilling from a technology perspective and terrifying from a security perspective.

## Prompt injection

The biggest potential risk given how OpenClaw works is prompt injection. 

Shared computing systems from the time of Unix in the 1970's until now have always had built-in protections and layers of security. Many of the original ideas created to keep operating systems secure (e.g., the sudo command) still work. 

The issue with the capabilities of a system like OpenClaw is an unprecedented level of autonomy at the user level with no human intervention. It's almost as if we have stripped computing down to the last wafer-thin layer of protection between the agent and the rest of the world. Know-how, skill and experience are not necessary to operate OpenClaw, just a text message.

The danger of prompt injection? If someone other than you can send text messages to your OpenClaw agent, the system has virtually no ability to discern an order coming from you versus an order from someone pretending to be you. A prompt injection could transfer your money to a new bank account, apply for a new line of credit, blackmail everyone in your contacts app. I am not capable of imagining the worst thing that could happen.

Prompt injections can also be indirect, meaning, your autonomous agent may encounter a hidden command or script on a web site you've instructed it to navigate, or in a malicious email attachment. It could be an instruction to dump all data OpenClaw can access: passwords, social security numbers, passport photos, personal document scans. 

## Can device management help?

In the next article in this series, we will investigate the ways in which threat detection and device management in Fleet can help secure the use of OpenClaw or detect it to prevent its use.  

<meta name="articleTitle" value="OpenClaw: Open for work?">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpounctual">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-02-04">
<meta name="description" value="Article series on managing devices running OpenClaw">
