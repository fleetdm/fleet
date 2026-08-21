# How Reed College brought its Windows fleet under management and reclaimed 20 hours a week

## The challenge: a Windows fleet nobody could see

Reed College's IT team supports faculty, staff, and students across more than 1,500 devices. The Macs had Jamf. The Windows PCs had nothing.

Reed had Intune bundled with its Microsoft E3 license, but it had never been set up. When the team finally sat down with it, the platform didn't fit the way they worked.

<div purpose="attribution-quote">

*We had Intune through our E3 licensing, but when we dug in, the complexity and learning curve were steep enough that the team agreed it wasn't going to work for us.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

Leaving those devices unmanaged had consequences the security team could measure. Nobody was managing software updates, so patch levels drifted, compliance goals went unmet, and the Windows fleet became a standing vulnerability. At a research institution, that abstract risk had a very concrete worst case.

<div purpose="attribution-quote">

*I didn't want to keep losing sleep over what would happen if an unmanaged PC with proprietary research data on it was stolen.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

The search for something better picked up momentum when a former colleague posted about their own migration to Fleet. The Reed team attended a GitOps workshop, brought in director-level stakeholders, and moved into a hands-on evaluation.

## Why Fleet

Choosing a platform at Reed is partly a cultural question. The college is deeply committed to academic freedom, and faculty expect control over their own machines. That expectation can sit in tension with IT's job of securing and standardizing the devices it manages. Whatever the team picked had to earn its place rather than be imposed.

<div purpose="attribution-quote">

*Open source was a huge draw. Reed has a long history with open source. We build a lot of our own tools in house and open source what we can. That ethos fits education. We're strong believers in equal opportunity for learning, and Fleet's open source philosophy lines up with that.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

That openness turned out to be practical, not just philosophical. During the evaluation the team ran into a capability they wanted that didn't exist yet. Instead of filing a ticket into a black box, they could see exactly where it stood.

<div purpose="attribution-quote">

*I could just search the public GitHub issues, found that it was already being worked on, and I could even see which sprint it was landing in. That kind of transparency told me this was a product team that actually listens to what customers are asking for.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

Reed also chose Fleet's managed cloud. The college was already working to reduce the number of on-premises servers it ran, and adding another one to stand up device management would have moved in the wrong direction.

By the end of the evaluation, the decision was straightforward. The IT team was aligned, the college's values were reflected in the product, and Fleet addressed the two most urgent gaps: visibility into the Windows fleet and automated software management to meet compliance requirements. The bake-offs the team had planned against other MDMs never became necessary.

## The solution

**Getting Windows under management for the first time.** With no existing MDM enrollment, there was no easy way to push the agent out, so the team used group policy to get devices enrolled. Enrollment took two to three months. New devices now land in a staging fleet, and Reed's hardware shop moves each one into the appropriate fleet as part of preparing it for its end user. Fleet-maintained apps keep Windows software current, and the team is moving toward automated patching to close the update gap that made Intune a liability.

**Answering questions in seconds instead of fighting the console.** The everyday difference shows up in visibility.

<div purpose="attribution-quote">

*With Intune, I struggled to navigate it to answer simple questions, like what software is on this device, and is that software actually up to date? Now I set up a policy or run a live query and get an answer immediately. Targeting devices with fleets and labels is genuinely easy. Visibility and taking action were both time-consuming before. Now they're not.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

**Compliance the security team can see.** Reed's security team defines the policies. IT monitors policy failures and remediates them, often by resending a configuration profile. Standardized controls like CIS benchmarks are next on the list. Logs ship to Security Onion, where the team builds the dashboards leadership relies on to understand Reed's security posture.

**Letting users help themselves.** Reed deployed Fleet Desktop and turned on self-service so people can resolve their own policy failures. The team published a transparency page explaining exactly what the agent does and why. They braced for pushback from an opinionated user base, and it never came.

## The results

The most immediate change was time.

<div purpose="attribution-quote">

*I'm spending about 20 hours a week less on managing PCs than I was with Intune.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

The bigger shift was visibility where there had been none.

<div purpose="attribution-quote">

*We had no insight into PC compliance before. Now I've got dashboards and reports to prove that we're at 90% device compliance, with 95% in our sights.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

And the worst case that used to hang over the unmanaged fleet now has an answer.

<div purpose="attribution-quote">

*Being able to reliably lock and wipe a device remotely is real peace of mind. I don't lie awake wondering what happens to research data on a stolen PC anymore.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

With Fleet, Reed College has:

<div purpose="checklist">

20 hours a week back for a team that had been managing PCs by hand

90% device compliance on Windows, up from no compliance visibility at all, with 95% as the next target

Remote lock and wipe on every managed PC, including the ones carrying research data

Self-service policy remediation in Fleet Desktop, adopted without the pushback the team expected
</div>

The team is candid that this is a journey rather than a finished state. Provisioning is still hands-on, and Reed hasn't moved to a zero-touch workflow yet. But the direction is clear, and a fully managed fleet that includes macOS, mobile, and Linux is on the horizon.

## Looking ahead

Reed plans to bring the IT team's own Macs into Fleet as a first step toward managing macOS there, and is already exploring declarative device management and Platform SSO.

Asked what they'd tell a peer at another institution weighing the move, Mason's answer wasn't about a feature at all.

<div purpose="attribution-quote">

*The transparency and the customer support. I can't say enough about how supported we felt throughout the whole process. There wasn't a moment when we didn't feel like Fleet had our back.*

**Mason Peressini**

Sr Systems Support Specialist, Reed College
</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="How Reed College brought its Windows fleet under management and reclaimed 20 hours a week">
<meta name="description" value="Reed College replaced an unmanaged Windows fleet with Fleet, reaching 90% device compliance and saving 20 hours a week of manual work.">

<meta name="publishedOn" value="2026-08-21">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="reed-college-logo-152x40@2x.png">
<meta name="quoteAuthorImageFilename" value="mason-peressini-120x120@2x.png">
<meta name="quoteAuthorName" value="Mason Peressini">
<meta name="quoteAuthorJobTitle" value="Sr Systems Support Specialist, Reed College">
<meta name="quoteContent" value="“The transparency and the customer support. I can't say enough about how supported we felt throughout the whole process. There wasn't a moment when we didn't feel like Fleet had our back.”">

<meta name="companyName" value="Reed College">
<meta name="companyInfo" value="Reed College is an independent liberal arts college in Portland, Oregon, known for its rigorous academics and its deep commitment to academic freedom. The college has a long history of building tools in house and open sourcing what it can.">
<meta name="companyInfoLineTwo" value="Reed's IT team supports faculty, staff, and students across more than 1,500 devices, and manages its Windows fleet in Fleet's managed cloud.">

<meta name="summaryChallenge" value="Reed College's Windows fleet was completely unmanaged. Intune came bundled with the college's Microsoft E3 license but had never been set up, and the learning curve was steep enough that the team ruled it out. Nobody was managing software updates, IT had no way to verify patch compliance, and a lost or stolen PC carrying research data could not be locked or wiped.">
<meta name="summarySolution" value="Fleet gave Reed's small IT team visibility and control over Windows for the first time, with group policy enrollment, a staging fleet that feeds devices into the right fleet during provisioning, Fleet-maintained apps for software updates, and self-service policy remediation in Fleet Desktop. Reed runs Fleet's managed cloud to avoid standing up another on-premises server.">
<meta name="summaryKeyResults" value="Reclaimed 20 hours a week previously spent managing PCs; Reached 90% device compliance on Windows, up from no compliance visibility at all, and is targeting 95%; Gained reliable remote lock and wipe for lost or stolen devices carrying research data; Enabled self-service policy remediation for end users, adopted without the pushback the team expected">
