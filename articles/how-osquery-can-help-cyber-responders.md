# How osquery can help cyber responders

![How osquery can help cyber responders](../website/assets/images/articles/osquery-for-cyber-responders-1600x900@2x.png)

At 3 a.m. on Sunday morning, Avery's phone starts vibrating. It's their SOC lead telling them about an in-progress ransomware attack. The entire next quarter is a blur of whiteboarding and crisis meetings. 

After three months of endless shifts, much of which is spent trying to tie together information from thousands of endpoints, Avery's team manages to save 90% of their data. A good result. But who cares? Management even pins some blame on Avery's team despite the C-suite ignoring warnings and delaying pen tests. 

An above-inflation pay raise? Forget about it. Anxious, disappointed, and still not sleeping, Avery, burned out, quits their job. Avery is a fictional character, but [a study from IBM](https://newsroom.ibm.com/2022-10-03-New-IBM-Study-Finds-Cybersecurity-Incident-Responders-Have-Strong-Sense-of-Service-as-Threats-Cross-Over-to-Physical-World) puts data behind the reality that thousands of men and women on the front line of cyber incident response know well. 

The vast majority (67%) of incident responders surveyed by IBM report feeling stress/anxiety due to their jobs. 30% get insomnia, the same number feel burned out, while 18% feel physical effects, and 17% get panic attacks. 

Working in cyber incident response is not suitable for your health. But cyber incident responders’ jobs can be made better. One actionable way to do this is by using osquery and decreasing the time it takes to respond and remediate cyber incidents - helping analysts get their lives back sooner. 


## Low visibility hurts incident responders 

In their report, IBM recommends two core actions for decreasing stress: creating detailed incident response plans and testing your incident response under pressure.

It's worth adding a third recommendation - centralize and simplify fleet management. 

During the investigation and data-gathering phase of an incident response effort, analysts must quickly gather information from across all their endpoints. Few can do it efficiently. 

Lack of visibility into IT security infrastructure is a severe challenge, [with almost two-thirds (62%)](https://www.intelligentciso.com/2022/06/13/global-organisations-concerned-digital-attack-surface-is-spiralling-out-of-control/#:~:text=Visibility%20challenges%20appear%20to%20be,cited%20as%20the%20most%20opaque.) of organizations reporting they have security blind spots that hamper their security efforts. 

The result is that when a series of alerts indicate a malicious incident, analysts can end up in the dark about large parts of their digital estate. They can be incredibly stressful and frustrating when an incident is a false positive and potentially devastating when a genuine attack is in progress and you can’t get to the bottom of it fast enough. 

For cyber incident responders, who are sometimes compared to firefighters, the building is burning down, and smoke is everywhere, but no one has a floor plan. 


## The case for unifying endpoint data

An unequal security landscape is hampering endpoint visibility. Most organizations cannot gather unified data about all of their endpoints on a single screen. 

For example, if an incident response team wanted to quickly figure out what files have been deleted from hundreds of different endpoints or which applications are running vulnerable versions of SSL, where would they start? 

According to [research from TrendMicro](https://www.intelligentciso.com/2022/06/13/global-organisations-concerned-digital-attack-surface-is-spiralling-out-of-control/#:~:text=Visibility%20challenges%20appear%20to%20be,cited%20as%20the%20most%20opaque.), the average organization can only see around 62% of its attack surface. The other 38% is opaque.

In a world where lateral movement is a feature in the majority of cyber attacks (and [96% of the time](https://www.intelligentciso.com/2021/04/09/expert-says-cisos-need-to-take-lateral-movement-seriously/) doesn't result in a SIEM alert), visibility gaps like these are a terrifying prospect for responders. Threat actors will almost always attempt to "land and expand," mainly using hijacked versions of legitimate tools or exploiting misconfigurations in interconnected cloud environments that bypass signature-based controls.

To cut analysts' stress levels and reduce their mean time to respond (MTTR), start by plugging their visibility gap. 


## Complex tool stacks are not helping

Growing tool stacks that consist of endpoint detection and response (EDR), endpoint protection platforms (EPP), and other point solutions organizations lean on to understand and secure their environments aren't necessarily improving visibility. 

Modern IT environments can feature a mix of on-premises and cloud endpoints across different flavors of Linux and Windows versions. Some devices will have a full suite of security agents installed; others will not. 

This means incident responders can get relatively granular insights into some parts of their environment. A modern Windows workstation with an AV+EDR agent like Windows Defender for Endpoint installed will be much easier to query than a legacy Linux server. 

The same is true for mobile devices. Very few organizations are confident in their processes to inventory various assets connected to their networks. Only 23% of the organizations we surveyed in our [state of device management report](https://fleetdm.com/reports/state-of-device-management) successfully enroll all or nearly all of their devices into their Mobile Device Management (MDM) system. 


## Simplifying endpoint visibility with osquery and Fleet

Incident responders’ jobs are much more manageable, and their companies are safer when they can query every device connected to their network. 

Open-source and deployable on macOS, Windows, Linux, and Chromebook devices and servers, osquery agents can meet their challenges.

Using Fleet as a centralized management interface to deploy, update, and manage osquery agents, incident response teams can ask questions about their endpoints and get real-time results on a single interface. 


<meta name="category" value="security">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-11-02">
<meta name="articleTitle" value="How osquery can help cyber responders.">
<meta name="articleImageUrl" value="../website/assets/images/articles/osquery-for-cyber-responders-1600x900@2x.png">
