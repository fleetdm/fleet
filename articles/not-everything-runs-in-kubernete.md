# Not everything runs in Kubernetes 

Dedicated infrastructure still matters. Most teams rely on bare-metal servers, virtual machines (VMs), and specialized hardware in addition to containers. These systems aren't going anywhere, and they require proper management.

Some teams use long-lived cloud instances like EC2 or Azure VMs for performance, compliance, or cost reasons. Others operate hybrid environments or manage hardware in factories, labs, and data centers.

Container security isn’t enough. We need to protect our dedicated assets.

These systems need updates, monitoring, and visibility. We need some way to know how they're configured, if they're still secure, and whether they still exist. If you can’t see what a system is doing, you can’t secure or maintain it.

At Fleet we built with these realities in mind from day one. Not only was it important to make Fleet easy to deploy anywhere (the server itself), it was also important to make sure people could enroll any part of their infrastructure, across legacy networks, VPNs, and computing platforms.

This flexibility isn’t a nice-to-have - it’s required for the majority of teams that manage systems that are not all containerized and cloud-native.

Infrastructure, whether on-prem or in the cloud (or on your desk), doesn’t manage itself. Whether it's a Linux server racked in a data center, a VM in the cloud, or a fleet of devices on a factory floor, we know companies need visibility and control features at scale. With so many diverse environments using Fleet, we made sure it's easy to stay on top of updates, no matter what security posture and system integrity rules you need to enforce.

This is why Fleet was designed to work across platforms, without forcing teams into a single deployment model or tooling stack.

Today, Fleet manages and secures millions of devices, from AWS servers, to containers, large gaming datacenters, supercomputer control nodes, factory robots, employee laptops, MrBeast's iPads, and more.

Let's look at some case studies:

#### Large gaming company enhancing server observability: 
<blockquote purpose="quote">
Fleet's extremely wide and diverse set of data allows us to answer questions that we didn't even know we had. On top of that, the experience is near instantaneous. Seconds to sort through billions of data points and return the exact handful that we need, with complete auditing and transparency. We're able to address reliability and compliance concerns without sacrificing a single point-of-a-percent of performance for our servers. All of this done consistently and continuously.
</blockquote>

#### Cloud-based data leader chooses Fleet for orchestration: 
<blockquote purpose="quote">
I wanted an easy way to control osquery configurations, and I wanted to stream data as fast as possible. No other solution jumped out to solve those things except for Fleet.
</blockquote>

#### Vehicle manufacturer transitions to Fleet for endpoint security:
<blockquote purpose="quote">
Fleet has become the central source for a lot of things. The visibility down into the assets covered by the agent is phenomenal.
</blockquote>

<meta name="articleTitle" value="Not everything runs in Kubernetes">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="authorGitHubUsername" value="zwass">
<meta name="category" value="announcements">
<meta name="publishedOn" value="2025-05-27">
<meta name="description" value="Why Fleet goes beyond Kubernetes to manage real-world infrastructure.">
