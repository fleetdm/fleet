# How healthcare AI company, Abridge, gets new hires productive in 10 minutes with Fleet

## The challenge

Abridge exists to give clinicians their time back. Using AI to handle the documentation and charting that pulls doctors away from patients and, too often, follows them home after a long shift. For a company whose entire premise is helping healthcare move faster and more safely, the tooling that runs its own devices had to meet the same standard. Before Fleet, it wasn't.

When Gabe Chan joined as a Staff IT engineer in early 2026, one of his first assignments made the gap obvious. Leadership wanted to know which employees were using a particular AI tool, a straightforward governance question at a health-tech company handling protected data. He couldn't answer it.

<div purpose="attribution-quote">

*There was just no way to pull that report. I ended up writing a hacky script to look through directories, and I still couldn't get my leadership team a real answer that I felt confident in.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

While their previous MDM worked well at a smaller scale, Abridge had grown to 400 devices by the end of 2025 and was planning to double in size the following year. The platform's limitations were showing. Complicating matters, HR owned the other tool, so IT didn’t have the permissions they needed to manage devices securely. Underneath the day-to-day friction was a risk problem that a healthcare company can't wave off.

<div purpose="attribution-quote">

*We've doubled in size since I joined. If we changed nothing, we simply wouldn't be able to manage our fleet. With PHI and PII compliance requirements to adhere to, not being able to detect something like shadow MCP usage across our devices is a risk that's too high for us to take.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

Having previously evaluated Fleet at another company, Gabe knew what he wanted and was trusted to make the call.

## Why Fleet

For Gabe, the decision came down to the ability to use GitOps workflows via infrastructure as code, having the power of visibility and querying during incident response, and using a platform built for the way modern, AI-accelerated teams actually work.

Fleet's open-source foundation and deeper integrations mattered too: native osquery, log shipping to Google Pub/Sub, and an easy plug-in with Claude to their repo gave the team room to build the workflows they wanted from day one.

Fleet's support was a differentiator in its own right. A Slack channel staffed by support engineers and solutions architects with real device management expertise, fast response times, and bug reports tracked openly in GitHub were influential factors in their decision.

## The solution

**Onboarding as a first impression.** For a company competing for engineering and clinical talent, a new hire's first day sets the tone. Abridge built a zero-touch onboarding flow with Fleet. A new hire receives their device through the hardware vendor, receives Okta credentials by email, and the zero-touch workflow does the rest: apps are installed, policies and profiles are applied, and within ten minutes the device is ready for use.

Engineers whose setups are notoriously heavier get a tailored version of the same flow.

<div purpose="attribution-quote">

*Setting up an engineer's laptop used to take us the whole day. Now we configure setup experience and automations that were built in coordination with Fleet’s support engineering and service delivery team. It's about 45 minutes to get an engineer ready on day one.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

An engineering leader reached out to Gabe to share that, “The work you've done to improve the default environment Engineers come into when they join Abridge is going to make an amazing difference in time-to-ramp and overall polish of the end experience.”

**Configuration as code.** The team now scopes configuration profiles and policies by device and user via GitOps automations, using pull requests and approvals, instead of targeting devices one serial number at a time in their previous MDM. Anyone on the team can propose a change; nothing ships without review.

**Security visibility, finally.** Abridge collects device data and pipes it into Google Pub/Sub, then runs policies and scheduled queries against it. For disaster-recovery or incident response scenarios, the team leans on live queries for instant information about the fleet.

<div purpose="attribution-quote">

*I can report on CVEs and tell security exactly which devices are affected by potential vulnerabilities. We can finally detect shadow MCPs in use. We went from 30 minutes to answering security questions during an active incident, down to 2-3 minutes.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

## The results

The migration was fast and, notably, quiet. ADE-enrolled devices transitioned effortlessly, and the feedback reached the top of the org.

<div purpose="attribution-quote">

*Users couldn't believe that the migration only took three or four minutes. I heard that consistently from VPs and above across the company.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

The advice Gabe would give to other IT teams considering a migration: plan to migrate devices first, layer profiles and policies afterward, and give users clear screenshots and short videos on what to expect during the process. Abridge kept things moving quickly with 30% of devices enrolled on day one, 60% within a week, and 100% at the end of the month.

Just as important for a lean team supporting a scaling company, the everyday cost of knowing what’s going on in your environment dropped sharply. In working with security, Gabe said,

<div purpose="attribution-quote">

*A query that used to mean scripting, echoing, and validating work took 30 minutes, and I still wasn't sure if I was providing an accurate answer to my leadership team. Now it takes two or three minutes, and I trust the results every time.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

An unexpected additional benefit for IT was that ticket volume eased as end users moved to self-service policy remediation and software updates via Fleet Desktop. Fleet-maintained apps replaced the clumsy app-update experience of the past, and users had visibility into their own device’s security posture.

<div purpose="attribution-quote">

*AI is going to take over. We're constantly told to leverage it and move five to ten times faster. MDM isn't new, but many legacy tools aren't keeping up. You have to choose something nimble enough to move at that speed.*

**Gabe Chan**

Staff IT Engineer, Abridge
</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="How healthcare AI company, Abridge, gets new hires productive in 10 minutes with Fleet">
<meta name="description" value="Abridge uses Fleet's GitOps workflows and real-time visibility to get new hires productive in 10 minutes and answer security questions in minutes.">


<meta name="publishedOn" value="2026-09-03">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">


<meta name="companyLogoFilename" value="abridge-logo-220x40@2x.png">
<meta name="quoteAuthorImageFilename" value="gabe-chan-120x120@2x.png">
<meta name="quoteAuthorName" value="Gabe Chan">
<meta name="quoteAuthorJobTitle" value="Staff IT Engineer">
<meta name="quoteContent" value="“Setting up an engineer's laptop used to take us the whole day. Now we configure setup experience and automations that were built in coordination with Fleet’s support engineering and service delivery team. It's about 45 minutes to get an engineer ready on day one.”">

<meta name="companyName" value="Abridge">
<meta name="companyInfo" value="Abridge builds generative AI for clinical conversations. It turns the clinician-patient dialogue into structured documentation, so care teams can focus on patients rather than on charting and billing. Abridge is used by more than 300 health systems and powers over 100 million conversations a year.">
<meta name="companyInfoLineTwo" value="It was named Best in KLAS for ambient AI in both 2025 and 2026 and recognized among TIME's Best Inventions and Forbes' AI 50. Learn more at abridge.com.">

<meta name="summaryChallenge" value="Abridge builds AI that turns patient conversations into clinical documentation. Its previous MDM was owned by HR, leaving IT unable to manage devices securely or answer basic security questions, like which employees were using a given AI tool. With PHI and PII compliance requirements, that lack of visibility was too high a risk to accept.">
<meta name="summarySolution" value="Fleet provides Abridge's IT team with infrastructure-as-code, GitOps workflows, and real-time visibility across the fleet. Device data flows into Google Pub/Sub, where policies and queries run against it, giving instant answers during incident response.">
<meta name="summaryKeyResults" value="New hires have a device ready in 10 minutes, down from a full business day; Engineer laptop setup takes 45 minutes, down from a full day; Incident response questions answered in 2–3 minutes, down from 30 minutes (about a 90% reduction); 30% of devices migrated on day one, 60% within a week, 100% within a month; Per-device migration took 3–4 minutes, with no disruption to end users; Full visibility into shadow AI and CVE exposure across the fleet; Ticket volume eased as users shifted to self-service remediation via Fleet Desktop; 700+ devices managed through infrastructure as code">
