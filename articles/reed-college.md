# How Reed College gained visibility into 300 unmanaged PCs and reclaimed 20 hours a week

## The challenge

Reed College's IT team supports faculty, staff, and students across more than 1,500 devices. As that environment grew, the team lacked the centralized tooling it needed to manage and report on the Windows fleet.

Reed had access to Microsoft Intune through its existing Microsoft licensing. After evaluating deployment options, the team decided a different approach would better fit its operational needs and staffing model.

<div purpose="attribution-quote">

*We had Intune through our E3 licensing, but when we dug in, the complexity and learning curve were steep enough that the team agreed it wasn't going to work for us.*

</div>

Those unmanaged devices had an immediate impact on the security team's goals. The team wanted stronger visibility into software update status and device compliance to support evolving security requirements.

<div purpose="attribution-quote">

*I didn't want to keep losing sleep over what would happen if an unmanaged PC with proprietary research data on it was stolen.*

</div>

The search for something better picked up momentum when a former colleague posted about their own migration to Fleet. The Reed team attended a GitOps workshop, brought in director-level stakeholders, and moved into a hands-on evaluation.

## Why Fleet

For Reed, there was a cultural dimension to deciding which solution would work best. Reed College is deeply committed to academic freedom, and faculty value control over their own machines. That value can sit in tension with IT's goals of observing, securing, and standardizing the devices under its management. Any solution the team chose had to earn its place, not just be imposed.

<div purpose="attribution-quote">

*Open source was a huge draw. Reed has a long history with open source. We build a lot of our own tools in house and open source what we can. That ethos fits education. We're strong believers in equal opportunity for learning, and Fleet's open source philosophy lines up with that.*

</div>

That openness turned out to be practical, not just philosophical. During the evaluation the team ran into a capability they wanted that didn't exist yet. Instead of filing a ticket into a black box, they could see exactly where it stood.

<div purpose="attribution-quote">

*I could just search the public GitHub issues, found that it was already being worked on, and I could even see which sprint it was landing in. That kind of transparency told me this was a product team that actually listens to what customers are asking for.*

</div>

Reed also chose Fleet's managed cloud. The college was already working to reduce the number of on-premises servers it ran, and standing up another one for device management would have moved in the wrong direction.

By the end of the evaluation, Fleet made the decision easy for Reed College. The IT team was aligned, the college's values were reflected in the product, and there was an immediate solution to their most critical gaps: visibility into the Windows fleet and automated software management to meet compliance requirements. Planned bake-offs against other MDMs never became necessary.

## The solution

**Getting Windows under management for the first time.** New devices now go through a staging process, then move into the appropriate fleet once they are prepared for their end user. Fleet-maintained apps keep Windows software current, and the team is moving toward automated patching, closing the update gap that made using Intune a liability.

**Answering questions in seconds instead of fighting the console.** The everyday difference shows up in visibility.

<div purpose="attribution-quote">

*With Intune, I struggled to navigate it to answer simple questions, like what software is on this device, and is that software actually up to date? Now I just set up policies or use a live query and get an answer immediately. Targeting devices with fleets and labels is genuinely easy. Visibility and taking action were both time-consuming before. Now they're not.*

</div>

**Compliance the security team can see.** The security team defines the policies. The IT team monitors policy failures and remediates them, with standardized controls like CIS benchmarks on the roadmap. Logs ship to Security Onion, where the team pulls the dashboards leadership relies on to understand Reed's security posture.

**Letting users help themselves.** Reed leaned into deploying Fleet Desktop and permitting self-service actions so people can resolve their own policy issues. The team published a transparency page explaining exactly what the agent does and why. They braced for pushback from an opinionated user base, but it never came.

## The results

The most immediate change was time.

<div purpose="attribution-quote">

*I'm spending about 20 hours a week less on managing PCs than I was with Intune.*

</div>

The bigger shift was visibility.

<div purpose="attribution-quote">

*We had very limited insight into PC compliance before. Now I've got dashboards and reports to prove that we're at 90% device compliance, with 95% in our sights.*

</div>

And the worst case that used to loom over the unmanaged fleet now has an answer.

<div purpose="attribution-quote">

*Being able to reliably lock and wipe a device remotely is real peace of mind. I don't lie awake wondering what happens to research data on a stolen PC anymore.*

</div>

With Fleet, Reed College has:

<div purpose="checklist">

Reduced time spent managing PCs by 20 hours per week

Gone from limited compliance visibility to 90% device compliance, targeting 95%

Reliable remote lock and wipe for lost or stolen devices

Self-service policy remediation for end users, reducing IT's ticket load
</div>

The team is candid that this is a journey, not a finished state. They're still hands-on during provisioning and haven't moved to a fully zero-touch workflow. But the trajectory is clear, and a fully managed fleet, inclusive of their macOS, mobile, and Linux devices, is on the horizon.

## Looking ahead

Reed plans to bring its IT team devices onto Fleet for macOS testing, and is already exploring declarative device management and Platform SSO, extending Fleet's reach further into the environment.

Asked what they'd tell a peer at another institution weighing the move, the answer wasn't about a feature at all.

<div purpose="attribution-quote">

*The transparency and the customer support. I can't say enough about how supported we felt throughout the whole process. There wasn't a moment when we didn't feel like Fleet had our back.*

</div>

<meta name="category" value="case study">
<meta name="articleTitle" value="How Reed College gained visibility into 300 unmanaged PCs and reclaimed 20 hours a week">
<meta name="description" value="Reed College gained visibility into 300 unmanaged Windows PCs with Fleet, reaching 90% device compliance and reclaiming 20 hours a week.">

<meta name="publishedOn" value="2026-08-21">
<meta name="authorGitHubUsername" value="n/a">
<meta name="authorFullName" value="Fleetdm">

<meta name="companyLogoFilename" value="reed-college-logo-133x40@2x.png">
<meta name="quoteAuthorImageFilename" value="mason-peressini-120x120@2x.png">
<meta name="quoteAuthorName" value="Mason Peressini">
<meta name="quoteAuthorJobTitle" value="Sr Systems Support Specialist, Reed College">
<meta name="quoteContent" value="“The transparency and the customer support. I can't say enough about how supported we felt throughout the whole process. There wasn't a moment when we didn't feel like Fleet had our back.”">

<meta name="companyName" value="Reed College">
<meta name="companyInfo" value="Reed College is an independent liberal arts college in Portland, Oregon, known for its rigorous academics and deep commitment to academic freedom. Its IT team supports faculty, staff, and students across more than 1,500 devices.">
<meta name="companyInfoLineTwo" value="Reed manages its Windows fleet in Fleet's managed cloud.">

<meta name="summaryChallenge" value="Reed College's Windows fleet was unmanaged, IT couldn't verify patch compliance, and a lost or stolen laptop had no way to be remotely locked or wiped.">
<meta name="summarySolution" value="Fleet gives Reed's small IT team visibility and control over Windows devices, with staged onboarding, automated software updates, and self-service policy remediation for end users. Reed runs Fleet's managed cloud to avoid standing up another on-premises server.">
<meta name="summaryKeyResults" value="Reduced time spent managing PCs by 20 hours per week; Went from limited compliance visibility to 90% device compliance, targeting 95%; Reliable remote lock and wipe for lost or stolen devices; Enabled self-service policy remediation, reducing IT's ticket load">
