# Fleet software attestation

At Fleet, we understand the importance of having a secure software supply chain.  Our core value of 🟣 [Openness](https://fleetdm.com/handbook/company#openness) extends to ensuring that our users can verify the provenance and authenticity of any Fleet software they install.  With that in mind, as of version 4.63.0 Fleet we will be adding [SLSA attestations](https://slsa.dev/) to our released binaries and container images.  This includes the Fleet and Fleetctl server software, the Orbit and Fleet Desktop software for hosts, and the osqueryd updates periodically downloaded by hosts.

## What is software attestation?

A software attestation is a cryptographically-signed statement provided by a software creator that certifies the build process and provenance of one or more software _artifacts_ (which might be files, container images, or other outputs). In other words, it's a promise to our users that the software we're providing was built by us, using a process that they can trust and verify. We utilize the SLSA framework for attestations which you can read more about [here](https://slsa.dev/).  After each release, attestations are added to https://github.com/fleetdm/fleet/attestations.

## Verifying our release artifacts

Any product of a Fleet release can be _verified_ to prove that it was indeed created by Fleet, using the `gh` command line tool from Github.  See the [`gh attestation verify`](https://cli.github.com/manual/gh_attestation_verify) docs for more info.

<meta name="authorGitHubUsername" value="sgress454">
<meta name="authorFullName" value="Scott Gress">
<meta name="publishedOn" value="2025-01-14">
<meta name="articleTitle" value="Fleet software attestation">
<meta name="category" value="guides">
