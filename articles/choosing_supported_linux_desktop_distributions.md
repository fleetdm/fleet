# Which Linux distributions should your organization support?

Once your organization decides to support Linux desktops, you will face a difficult question: which Linux distributions will you support? Answering this question can be overwhelming, even if your IT team has Linux knowledge. The Linux ecosystem is vast. The staggering variety of distributions, software, and configurations can make it difficult to narrow down your choices.

The answer to this question will look different for every organization. However, there are common questions that you can ask to accurately assess your needs. Once you have a cultural and technical understanding of these needs, you will be ready to choose your supported distros.

In this article, we will review the challenges of supporting the vast Linux ecosystem. We will then discuss the cultural and technical questions that you should ask when choosing supported distributions.

## The Linux ecosystem

The Linux ecosystem is highly heterogeneous when compared with Windows and Mac environments.  Nearly every aspect of the operating system, from the package management software to the network stack, can be heavily customized. Even an activity as simple as host resolution is replete with variety: systemd-resolved, dnsmasq, and plain resolvconf with external DNS servers can be used to look up hostnames.

The Linux kernel is the stable core of the ecosystem. Distributions build on this core to provide a complete desktop or server environment. Distributions are opinionated about the default software and configuration available to their end-users. Users can further customize the distribution according to their needs.

There are literally hundreds of Linux distributions, and you cannot practically support all of them. The goal of a Linux MDM project is to find the most appropriate distributions to support within your organizational constraints. This involves a conversation with your end users and compromises on both sides.

But first, it’s important to understand the cultural and technical constraints that will guide your evaluation.

## Cultural considerations

The Linux ecosystem has a distinct culture that goes beyond technical requirements. It also involves a cultural understanding of their needs and how they work. Failing to grasp these cultural concerns will lead to friction in your Linux MDM project.

Below are key questions to consider when evaluating your organization’s Linux culture. The answers to these questions will guide your decision about supported distributions.

**What distributions are your users already using?**

This is an obvious first question, but it’s an important one. Your organization may have an unmanaged Linux landscape, so survey your end users. Determine the distributions that they currently use. You may find that everyone has already self-standardized on a common distro, such as Ubuntu.

Alternatively, you may not yet be supporting any Linux in your environment. In that case, work with your users to understand the distributions they would like to use when given the opportunity. Be sure to understand the comfort level that users have with different distros. It’s quite common for Linux users to be proficient with multiple distributions, even if they have a preferred setup.

**Are your users open to a different distribution?**

Linux users are an infamously religious bunch when it comes to their desktop environments. However, you may find that they are surprisingly open to change. They may be willing to use a different distribution on their “work computer”.

Your initial investigation may find some particularly exotic distributions (this author personally prefers Void Linux). Ask your users if they would be open to using a distro that your team is willing to support. Finding reasonable ground to compromise on is an important part of any workstation management strategy, and Linux is no different.

**Is your organization willing to compromise? If so, how?**

You may find that some users have non-negotiable requirements, whether for a particular software distribution or a specific configuration. For example, a developer may be working on a lightweight, embedded product. They might insist on maintaining a similarly lean personal computer.

Determine if your organization is willing to compromise with those users and what that process will look like. Will you accept any explanation for a non-standard configuration? Or will you require written justifications that are approved by upper management? Will your IT team provide any support for custom configurations, or will the end user be on their own?

## Technical considerations

Ultimately, your supported distributions must meet your technical requirements. It’s important to understand these requirements early in the device management journey. Otherwise, you risk wasting time and accruing technical debt. 

Technical requirements vary wildly between organizations. However, there are some common questions to consider when evaluating supported Linux distributions.

**Does the chosen distribution support the environment your users need?**

For example, if your developers need Docker, then you must ensure that it can easily work on your chosen distribution. Does Docker provide official repositories for the distribution, or will that management burden fall on your IT team?

Be particularly conscious of vendor-specific software, such as VPNs or security software. These products do not always provide wide support for the Linux ecosystem. You may need to discuss these situations directly with those vendors or find a workaround if you must support a particular distribution.

**Does your management solution support the distributions you need?**

In an ideal world, you would first determine the distributions that your organization will support. Then, you would find the device management solution that also supports all of them. The reality is a bit different. It represents a compromise between your organizational constraints and the ecosystem of available solutions.

Your goal is to find a device management system that supports as many of your desired distributions as possible. For example, Fleet supports Debian, RedHat, Arch, and openSUSE. This gives users a variety of choices: Ubuntu and CentOS are very popular in the enterprise space, and Arch is a favorite of highly technical Linux gurus.

The distribution constraints imposed by your device management solution will also guide the compromises you make with your end users. You might need to make exceptions or take a hard stance on a distribution that you can’t support.

**How technically familiar is your desktop management team with a particular distribution?**

Your IT team is ultimately responsible for supporting the devices in your environment. They must be comfortable with the list of distributions that you will support. In some cases, these teams have little Linux experience, or their experience might be concentrated in server management.

Be sure to understand your team’s current capabilities and willingness to learn. It can be overwhelming for them to support a variety of distributions and configurations. Rather, you may choose to officially support a single distribution and plan to expand support as you gain expertise.

Similarly, consider the software and configurations that you will support. Do they have high-quality documentation and an active community to help you when you have questions? Or will your team be shouldering a heavy burden for niche software?

A Linux-native MDM solution, such as Fleet, can greatly ease this burden. However, your team must still have a base of knowledge to leverage the solution and troubleshoot when things go wrong.

**Should workstations have the same distribution as production server environments?**

Most organizations use containers, virtual machines, or remote development platforms. This reduces issues caused by mismatches between development and production environments. However, some product teams still need a bare-metal experience. Sometimes, developers or systems administrators might simply prefer to use an operating system that matches the production environment. This allows them to better understand the nuances of the OS.

This type of requirement can guide your choice of supported Linux distributions. If your production environment is hosted on CentOS, then you might prefer to only support CentOS as your chosen distro. This can also be a matter of team expertise: if the entire organization is proficient with a particular distribution, then it can be cheaper and easier to stick with that distro.

## Wrapping up

Determining the Linux distributions that you will support involves balancing the cultural and technical needs of your users and your organization. Taking the time to understand these needs will set you up for success in the Linux MDM domain.

Once you understand these needs, you can research and develop a Linux device management solution. Ideally, this solution will have native Linux support for popular distributions. This will ensure that the adoption of Linux devices enables your users and IT teams instead of hindering them.

To learn more about Fleet or to get a demo [contact us](https://fleetdm.com/contact).

<meta name="articleTitle" value="Which Linux distributions should your organization support?">
<meta name="authorFullName" value="Anthony Critelli">
<meta name="authorGitHubUsername" value="acritelli">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-18">
<meta name="description" value="Not all Linux distros are equal for enterprise IT. Learn how to evaluate cultural and technical factors to choose the right distributions to support.">
