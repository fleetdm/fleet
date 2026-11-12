# Put Microsoft Store apps on Windows with a script-only package

_Fleet runs scripts as SYSTEM, and winget can't install Store apps as SYSTEM. A Microsoft service can, and it installs the app for every user on the host._

## Key takeaways

- **winget alone won't work from Fleet.** Fleet runs Windows scripts as SYSTEM, and the `msstore` source refuses to install as SYSTEM.
- **A Microsoft service installs Store apps for every user.** A short PowerShell script calls it as SYSTEM, with no logged-in user and no scheduled task.
- **One script-only package per app.** The install script and uninstall script differ only by the Store ID, so adding an app is a pull request.
- **Built-in support is coming.** Fleet will [deploy Microsoft Store apps natively](https://github.com/fleetdm/fleet/issues/43493). Until then, script-only packages are the way.

<a purpose="cta-button" href="https://fleetdm.com/infrastructure-as-code">See it managed as code</a>

[Fleet 4.89.0](https://fleetdm.com/releases/fleet-4.89.0) added an uninstall script to script-only packages. That makes a `.ps1` file enough to install and remove a Microsoft Store app, with nothing to host.

## Why winget alone fails as SYSTEM

Fleet's agent runs Windows scripts as SYSTEM. Microsoft's [winget troubleshooting documentation](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) says the winget CLI isn't supported in the system context. Even when you find the binary, the `msstore` source rejects the install with "Device wide install for msstore type is not supported under admin context."

Asking for `--scope machine` doesn't help either. For a Store package, it [installs the app for the SYSTEM account](https://github.com/microsoft/winget-cli/issues/4748), which looks like success and leaves nothing for your users.

## Install for every user with a Microsoft service

> **TODO (@marko-lisica):** Name the Microsoft service, explain how the script uses it, and link to Microsoft's docs.

The install script runs as SYSTEM, which is how Fleet already runs it. It installs the app for every user on the host, so it works at the login screen, on shared computers, and for users who sign in later.

Only the Store ID changes from app to app. Find it with `winget search --source msstore "<app name>"`, or copy it from the end of the app's URL on [apps.microsoft.com](https://apps.microsoft.com).

## Wire it into GitOps

Save the install and uninstall scripts in your GitOps repo, then add the package to the fleet where you want it:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
```

Each app is two files and a YAML block, so adding or removing one is a pull request. Reviewers see exactly what will run, and the catalog is auditable and reversible.

## See it live

- Follow the [step-by-step guide](https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide) to add your first Store app.
- Read the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for the full detail on script-only packages and uninstall scripts.
- Get a demo: [fleetdm.com/contact](https://fleetdm.com/contact).
- Join a free GitOps workshop: [fleetdm.com/workshops](https://fleetdm.com/workshops).

_Managing devices as code, one pull request at a time. Start with the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files) or [talk to us](https://fleetdm.com/contact)._

<meta name="articleTitle" value="Put Microsoft Store apps on Windows with a script-only package">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Use a Fleet script-only .ps1 package and a Microsoft service to install Microsoft Store apps on Windows hosts for every user.">
