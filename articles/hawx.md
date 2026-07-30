# How Hawx gets field technicians productive on hour one with Fleet

## The challenge

For Hawx, a technology-first pest control company, a field technician's phone isn't an accessory. It's how the job gets done. A technician who can't get their device provisioned can't be dispatched, log a service, or generate billable work. So when device onboarding stalls, it isn't an IT inconvenience, it's lost service capacity.

That made Hawx's seasonality the crux of the problem. Hiring ramps up hard in the warm months, bringing in droves of contract technicians, then slows in winter, which means a constant roller coaster of device onboarding and offboarding. With Jamf, these workflows hit a wall.

<div purpose="attribution-quote">

*The MDM screen held up every new technician during onboarding. You couldn't do anything with the phone until the person receiving it logged in, and nobody knew their username or password. We'd get calls from their personal phones blowing up the helpdesk. At the start of the summer season, all we did was troubleshoot phones.*

</div>

The friction wasn't limited to onboarding. Even routine support calls were slow, because simply identifying who was holding a given device was difficult. In Jamf, associating users with devices meant CSV uploads and manual matching.

With a renewal on the horizon, the team had a reason to look for something better.

<div purpose="attribution-quote">

*When a field technician calls in, they previously had to read off their phone number or dig the serial number out of settings. Often, we had to walk them through where to find their serial number in settings. That's five or ten minutes a call, determining which device and user we're working with before even starting to troubleshoot their issue, and we take five to ten of those a day all season long. With Fleet, all we need is their name.*

</div>


## Why Fleet

Hawx evaluated two other vendors alongside Fleet. Fleet's open-source code base and managed cloud option were influential, but they weren't the deciding factors.

<div purpose="attribution-quote">

*The slam dunk for us was that Fleet gave us control over the phone itself. We can run the device the way we want. Nobody else we looked at could offer that.*

</div>

That control, paired with reliable end-user association, was what made it possible to automate the seasonal enrollment and offboarding that had been the team's biggest time commitment. There was no waiting on a vendor to build a trigger Hawx needed. Pairing Fleet with the automation platform Tines, Hawx tailors onboarding to each individual field technician or corporate employee, then leans on its identity provider to do the rest. Hawx drives all of it through Fleet's API rather than the Fleet UI.

## The solution

**Onboarding that runs itself.** The old process required an IT admin to log in to the console, find the device, associate the correct user, and manually move it into the right group. The new flow is hands-off. A device automatically drops into a default fleet. Once the technician follows the instructions in their welcome email and connects through Okta, Hawx pulls their user association and Fleet handles the rest. When the device moves into the appropriate fleet, it receives the required profiles and policies, and Fleet installs the apps. No console babysitting required.

**Offboarding without the scramble.** The exit path is just as automated. When a termination notice comes through, Fleet and Tines locate the phone and act on it with logic tuned to the device owner's role.

<div purpose="attribution-quote">

*When a termination notice comes through, Fleet and Tines go find that phone and wipe it. It stays associated with the user and in its fleet until we assign someone new. For senior roles like a GM, we lock it instead of wiping it just in case we need to retrieve anything from that device.*

</div>

## The results

The migration took about a month. The payoff showed up immediately where it had hurt most: start-of-season onboarding support requests are gone.

The everyday support math improved too. The five to ten minutes once lost to identifying a caller's device before troubleshooting could even begin, across five to ten calls a day, is now down to seconds.

With Fleet, Hawx has:

<div purpose="checklist">

New technicians productive from hour one during peak hiring season

Onboarding and offboarding that run automatically, wiping or locking devices based on role

Up to ten minutes saved on every support call, with device and user identification down to seconds

A full migration off Jamf completed in one month
</div>

For Loren, success is measured in silence.

<div purpose="attribution-quote">

*The way we know Fleet is working well and that we made the right choice is that leadership isn't hearing from branch general managers that technology is a blocker to technician onboarding. No news is good news, and that's the peace Fleet brings me.*

</div>

The worry that used to define the start of every season, technician onboarding and offboarding, simply isn't a worry anymore.

<meta name="category" value="case study">
<meta name="articleTitle" value="How Hawx gets field technicians productive on hour one with Fleet">
<meta name="description" value="Hawx automated seasonal iOS onboarding and offboarding with Fleet, Tines, and Okta, eliminating start-of-season helpdesk floods in one month.">

<meta name="publishedOn" value="2026-07-29">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="hawx-logo-150x40@2x.png">
<meta name="quoteAuthorImageFilename" value="loren-farr-120x120@2x.png">
<meta name="quoteAuthorName" value="Loren Farr">
<meta name="quoteAuthorJobTitle" value="IT Manager, Hawx">
<meta name="quoteContent" value="“The slam dunk for us was that Fleet gave us control over the phone itself. We can run the device the way we want. Nobody else we looked at could offer that.”">

<meta name="companyName" value="Hawx">
<meta name="companyInfo" value="Hawx Pest Control provides residential and commercial pest control across more than a dozen states. Its model is built on prevention and consistency, backed by a technology-first approach, designed to keep homeowners informed about exactly what's happening at their property.">
<meta name="companyInfoLineTwo" value="A three-person IT team manages roughly 500 iOS devices in Fleet's managed cloud, supporting a field technician workforce that expands sharply every summer.">

<meta name="summaryChallenge" value="Hawx hires droves of contract pest control technicians every summer, and a technician who can't get their phone provisioned can't be dispatched. With Jamf, phones were stuck on the MDM screen until the technician logged in, and nobody remembered their credentials, flooding the helpdesk. Identifying which device belonged to which technician took five to ten minutes per call.">
<meta name="summarySolution" value="Fleet gives Hawx direct, programmatic control over every device through its API. Paired with Tines and Okta, Fleet automates onboarding and offboarding end to end, moving a technician's device into the right fleet with the correct profiles, policies, and apps the moment they verify their identity.">
<meta name="summaryKeyResults" value="New technicians productive from hour one during peak hiring season; Onboarding and offboarding that run automatically, wiping or locking devices based on role; Up to ten minutes saved on every support call, with device and user identification down to seconds; A full migration off Jamf completed in one month">
