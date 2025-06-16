# Fleet software attestation

As of version 4.63.0 Fleet added [SLSA attestations](https://slsa.dev/) to our released binaries and container images.  This includes the Fleet server, [fleetctl](https://fleetdm.com/docs/get-started/anatomy#fleetctl) command-line tool (CLI), and Fleet's agent (specifically the [Orbit](https://fleetdm.com/docs/get-started/anatomy#fleetd) component).

## What is software attestation?

A software attestation is a cryptographically-signed statement provided by a software creator that certifies the build process and provenance of one or more software _artifacts_ (which might be files, container images, or other outputs). In other words, it's a promise to our users that the software we're providing was built by us, using a process that they can trust and verify. We use the [SLSA framework](https://slsa.dev/) for attestations.  After each release, attestations are added to https://github.com/fleetdm/fleet/attestations.

## Verifying a release

Any Fleet release can be _verified_ to prove that it was indeed created by Fleet, using the `gh` command line tool from Github.  See the [`gh attestation verify`](https://cli.github.com/manual/gh_attestation_verify) docs for more info.

After downloading the [Fleet server binary](https://github.com/fleetdm/fleet/releases), here's how to verify:

```
gh attestation verify --owner fleetdm /path/to/fleet
```

Verify the [fleetctl binary](https://github.com/fleetdm/fleet/releases) (CLI):

```
gh attestation verify --owner fleetdm fleetdm /path/to/fleetctl
```

Currently, you can verify Fleet's agent (fleetd) on macOS and Linux. To verify, after installing fleetd on a macOS or Linux host, run this command::

```
gh attestation verify --owner fleetdm /usr/local/bin/orbit
```

<meta name="authorGitHubUsername" value="sgress454">
<meta name="authorFullName" value="Scott Gress">
<meta name="publishedOn" value="2025-01-14">
<meta name="articleTitle" value="Fleet software attestation">
<meta name="category" value="guides">
