# Enroll BYOD iOS/iPadOS hosts

Fleet supports two ways to enroll personal (BYOD) iPhones and iPads:

- **Account-driven User Enrollment**: the end user signs in to their Managed Apple Account in **Settings**. Work and personal data stay separate, and Fleet only manages work apps and data. The host shows as **On (personal)**. For set up and end user instructions, see [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account). _Available in Fleet Premium._
- **Profile-based enrollment**: the end user opens an enrollment link and installs Fleet's enrollment profile. The host shows as **On (manual - personal)**, and IT can't wipe it or lock the end user out. The steps are below.

> Neither works if [Allow only Apple Business enrollments](https://fleetdm.com/guides/apple-mdm-setup#turn-on-mdm-on-a-host) is on. Only devices assigned in Apple Business can enroll.

Fleet only collects software inventory for apps installed through Fleet. Built-in apps (e.g. Calculator) and apps installed by the end user aren't included.

## Profile-based enrollment

1. In Fleet, head to **Hosts** and select the [fleet](https://fleetdm.com/guides/fleets) for these hosts.
2. Select **Add hosts**, then the **iOS & iPadOS** tab. Select **Personal (BYOD)** and copy the enrollment link.
3. Share the link with your end users. It walks them through downloading and installing Fleet's enrollment profile.

> To enroll company-owned iPhones and iPads that aren't in Apple Business, select **Company-owned (fully-managed)** instead. These hosts show as **On (manual)**, and IT can wipe them and enforce all MDM restrictions.

> Each fleet's link includes the fleet's enroll secret, which assigns hosts to the right fleet. If the enroll secret is wrong, end users can still download the profile, but enrollment fails with a 403 error.

<meta name="articleTitle" value="Enrolling BYOD iPad/iOS devices in Fleet">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-20">
<meta name="description" value="Choose account-driven or profile-based enrollment for personal (BYOD) iPhones and iPads, and enroll them in Fleet.">
