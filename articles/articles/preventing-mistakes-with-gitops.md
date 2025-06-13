## Introduction

All SysAdmins have been there. It’s Friday afternoon - you make a few wrong clicks in your MDM. All of a sudden, devices that were in a specific team granting users access to your internal network have a configuration profile revoked. Just as you’re ready to sign off for the weekend you start getting Slack messages and realize your mistake. 😳

It’s a tough lesson to learn. 

But, what if there was a way, through GitOps and change management best practices, you could avoid all this? A modern change management methodology typically reserved for developers has now arrived for your IT team. 🛬

## What is “GitOps”?

GitOps is an operational framework that takes best practices used for application development such as version control, collaboration, compliance, and continuous integration / development, and applies them to device management automation. Key attributes of this system:

- Configurations are declaritively defined in a repo (such as GitHub or GitLab)
- Configurations represent a single source of truth for system state
- Automated synchronization between git and live infrastructure
- Continuous reconciliation to maintain system state
- Immutable configuration is managed through pull requests and code reviews

The ultimate goals of this approach? Improve reliability, reduce errors, and enable consistent, auditable management of your device infrastructure. 

## Getting Started 

Fleet publishes a starter template that we recommend checking out (available for both GitHub and GitLab.) 

> In this article we will be using GitHub but the general principles are the same. 

Clone the starter repo: `https://github.com/fleetdm/fleet-gitops` and create your own repo to which you will push code. 

> In a production environment, it is best to protect the `main` branch and only allow merging after a code review is conducted. It can be modified if needed, but, by default the apply action will run whenever code is committed to `main`.

An important benefit of GitOps is the ability to store all your environment secrets in GitHub - encrypted and protected from view. With the correct configuration, this prevents tampering and leaks.

Add the `FLEET_URL` and `FLEET_API_TOKEN` secrets to your new repository's secrets. If you’re working out of the template, also add `FLEET_GLOBAL_ENROLL_SECRET`, `FLEET_WORKSTATIONS_ENROLL_SECRET` and `FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET`.

This can be adjusted depending on how you want to leverage Teams and team names.

## A Typical GitOps Workflow

We will start with a traditional workflow to demonstrate the process used to commit changes to your Fleet instance. In this example we are adding a passcode policy for Macs by setting the minimum length to 12 characters. 

> For all examples in this article we will be using the GitHub Desktop app to do commits. Using `git` in the terminal will of course also work. Use whatever you’re most comfortable with.

![gif-1](../website/assets/images/articles/preventing-mistakes-1-711x385@2x.gif)

Here, after making changes to the `passcode.json` file, it has been added to the Team we are configuring under the `macos_settings` section.

![gif-2](../website/assets/images/articles/preventing-mistakes-2-480x270@2x.gif)

GitHub Desktop will automatically pick up changes. You can review each file and make commit comments. If all looks good, push your changes to the working branch.

![gif-3](../website/assets/images/articles/preventing-mistakes-3-711x385@2x.gif)

We create a PR to bring this change into the `main` production branch. In this example, branch protections are off so I can merge right to `main` but further on in the article this will change. 

## GitOps: The way it was meant to be

Another benefit of a GitOps approach is the ability for members of a team to review changes before they are applied in production. This encourages collaboration while ensuring all modifications to state are following best practices and compliance. In addition, if something breaks (which is inevitable) you have a ‘snapshot’ or point in time with a known working state to which you can easily roll back.

![gif-4](../website/assets/images/articles/preventing-mistakes-4-480x270@2x.gif)

The newest version of macOS is released and an engineer on your team wants to push a change to require an update of all hosts in the Workstations team. The IT engineer creates a branch to work from and makes the necessary changes, including setting a new target version and deadline.

```
macos_updates:
    deadline: "2025-02-15"
    minimum_version: "15.4.1"
```

Merging is blocked until a member of the team reviews and approves the changes. 

![gif-5](../website/assets/images/articles/preventing-mistakes-5-480x270@2x.gif)

Our IT manager is listed as the approver for these changes. The approver is notified of a pending PR for review. Is there a problem with some of the changes? Our engineer accidentally put in a version string that is not yet available. This will cause issues for our users when they try to update. The fix? Tag the engineer with some feedback and request changes to be made and re-committed. 

![Pr Approval](../website/assets/images/articles/pr-approval-921x475@2x.jpg)

After our engineer has updated code from the review, the approver can do a final review, approve and let the engineer merge this branch into `main` to trigger the apply workflow. This will push the changes into the production environment. ✨

![Pr Approval](../website/assets/images/articles/pr-approval-2-933x483@2x.jpg)

## GitOps mode in the UI
Fleet supports locking down the UI with [GitOps mode](https://fleetdm.com/guides/articles/gitops-mode), which prevents manual updates to any features or
settings configurable with GitOps.

## Conclusion

By adopting GitOps for device management, your team's work becomes observable, reversible and repeatable while automating your device configurations. Instead of making changes manually and risking unintended consequences, you gain a reliable, auditable workflow where every modification is reviewed, approved, and tracked.

This approach reduces human error and fosters teamwork. Whether you're enforcing security policies, managing OS updates, or deploying configuration changes, GitOps ensures consistency and control helping you avoid those last-minute Friday afternoon mishaps. 😥

Want to know more about Fleet's comprehensive MDM platform in code? Visit fleetdm.com and use the 'Talk to an engineer' [link](https://fleetdm.com/contact).

<meta name="articleTitle" value="Preventing Mistakes with GitOps">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-02-12">
<meta name="description" value="Use GitOps to manage your infrastructure in code and prevent mistakes">
