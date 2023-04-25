# Does osquery violate the New York employee monitoring law?

![OSquery image](https://fleetdm.com/images/articles/osquery-a-tool-to-easily-ask-questions-about-operating-systems-cover-1600x900@2x.jpg)

**TL;DR:** A new law requires private-sector employers based in the state of New York who participates in monitoring activities to provide written notice to all employees. However, this law does not apply to osquery (and, by extension, Fleet) because a) osquery does not use monitoring processes that would fall under “employee monitoring” as defined by this law and b) osquery meets the exemption criteria provided by this law. 

## On May 7, 2022, a new employee electronic monitoring law was enacted.

The legislation ([A.430/S.2628](https://legislation.nysenate.gov/pdf/bills/2021/S2628)) requires private sector employers (regardless of their size or type) in the state of New York to give employees written notice if they are being monitored or their emails, phone conversions, and/or internet usage is being intercepted in any way. 

This written notice must be:

- Shared with employees upon hiring, and receive their acknowledgment either electronically or in writing. 
- Displayed in a “conspicuous place” visible to all employees. 

In New York, monitoring employees without giving them notice now means breaking the law. 

While the law does not allow employees to directly take legal action for infringements (i.e., a provision for a private right of action), it can be enforced by New York’s Attorney General (AG). 

The law gives the AG the right to fine offenders for each violation of the law and scale fines based on the number of offenses reported. First-time offenders can be fined up to $500, with the fine increasing to $1,000 for the second violation and $3,000 for all subsequent violations.

One important exception to the law is that monitoring processes performed for the purpose of system maintenance and/or protection are exempt from this law.

## Does osquery fall under the New York employee monitoring law?

To determine whether osquery use is covered by the new employee monitoring law, let’s look at two important factors:

- Definition of “employee monitoring” under the new law.
- Exemptions to the new law.

## Defining “employee monitoring”

The first of these factors concerns what exactly the new law’s authors meant with the term “employee monitoring.” 

According to the current wording of the law, employee monitoring processes include the monitoring or interception of:

“telephone conversations or transmissions, electronic mail or transmissions, or internet access or usage of or by an employee by any electronic device or system, including but not limited to the use of a computer, telephone, wire, radio, or electromagnetic, photoelectronic or photo-optical systems.” 

By default, osquery does not monitor any of the above. For example, osquery has repeatedly rejected user requests to make browser history available. 

In [its documentation](https://github.com/osquery/osquery/blob/bf2b464301d96b0033a21978faaf3f41719ae04d/docs/_docs/faq.md), osquery clearly states:

“We include a "non-goal" of exposing sensitive information like browsing history within tables.” 

Osquery [does not have access to](https://fleetdm.com/transparency) user email or text messages, screen content, keystrokes, mouse movements, or webcams and mics. As such, osquery use does not fit the definition of “employee monitoring” as defined by this law and is, on this basis, exempt from this law. 

## Exemptions

Osquery also meets the law’s exemption criteria. 

The law exempts employee monitoring if it is done solely for the purposes of system maintenance and security: 

“The provisions of this section shall not apply to processes that are designed to manage the type or volume of incoming or outgoing electronic mail or telephone voice mail or internet usage, that are not targeted to monitor or intercept the electronic mail or telephone voice mail or internet usage of a particular individual, and that are performed solely for the purpose of computer system maintenance and/or protection.” 

It’s true that the law does not provide a definition for “computer system maintenance” or “computer system protection.” But since osquery’s core use cases are system status monitoring and security, we can still assume that the exemption applies. 

Break down the meaning of “computer system maintenance” and “computer system protection” further, and osquery’s exempt status looks more certain. To see why, refer to the National Institute of Standards and Technology (NIST) glossary’s definition of “computer system maintenance” and “security protections.” 

## Computer system maintenance

According to NIST, computer system “[maintenance](https://csrc.nist.gov/glossary/term/maintenance#:~:text=1%2C%20NIST%20SP%20800%2D66,or%20restores%20its%20operating%20capability.)” is:

“Any act that either prevents the failure or malfunction of equipment or restores its operating capability.” 

Computer system maintenance is an important capability within osquery. 

IT teams commonly use osquery to keep an eye on enrolled device health and performance, including the processors used, battery health, amount of memory installed, and software details. 

This allows them to spot and fix any potential issues before they cause system failure or malfunction.

As a result, the law’s system maintenance exception applies to osquery. 

## Computer system protection 

For “computer system protection,” let’s use NIST’s “[security protections](https://csrc.nist.gov/glossary/term/security_protections)” definition, which is as follows:

“The management, operational, and technical controls (i.e., safeguards or countermeasures) prescribed for an information system to protect the confidentiality, integrity, and availability of the system and its information.”

Osquery has a strong security use case. In fact, it’s the first thing you see when you land on osquery’s homepage. 



The monitoring tool was developed to make endpoint and server monitoring easier for security teams and is used by organizations to detect suspicious account log-ins, identify malicious applications, and find security misconfigurations, among other things. 

Consequently, the law’s computer system protection exemption also applies to osquery. 

## Fleet and the new employee monitoring law

Osquery does not fall under the new employee monitoring law. And because Fleet is built on osquery, this law does not impact our device management platform either.

At Fleet, privacy and transparency are crucial to us. 

We urge employers to notify employees that their devices are monitored using osquery and Fleet, even if the New York employee monitoring law does not require employers to do so.

That’s why we built Fleet Desktop (available on Windows, Linux, and macOS) to include an icon that automatically lets users know they’ve been enrolled in Fleet. Clicking on the icon will bring users to [Fleet’s transparency page](https://fleetdm.com/transparency), which shows exactly what kind of information osquery and Fleet can see about them and their devices.

<meta name="category" value="security">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-04-18">
<meta name="articleTitle" value="Does osquery violate the New York employee monitoring law?">
