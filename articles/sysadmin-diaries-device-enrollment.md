# Sysadmin diaries: device enrollment

![Sysadmin diaries: passcode profiles](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

As sysadmins, we're tasked with the critical responsibility of securely managing devices within our organizations. Central to this endeavor is device enrollment, which lays the foundation for effective device management and security protocols. The question, "What happens if an employee removes Fleet?" was posed. And, as with all great answers, it starts with, "Well, it depends." It depends on device enrollment. To answer this question, let's look into two primary device enrollment methods, Automatic Device Enrollment (ADE) and Bring Your Own Device (BYOD), exploring their nuances, strengths, and considerations for sysadmins.


## ADE

Automatic Device Enrollment, often called DEP, empowers sysadmins to seamlessly maintain control over corporate devices. With ADE, devices are automatically provisioned and enrolled into management systems like Fleet. This automated process ensures that devices are configured with predefined settings and security measures, enhancing overall device security. ADE restricts users from unenrolling devices, providing sysadmins with the assurance of continued control even in scenarios such as device loss or theft. Removing Fleet in this scenario merely removes the `fleetd` agent, but we can still send MDM commands to the host, such as remote lock or wipe as long as the MDM enrollment profile remains untouched. Without the `fleetd` agent, the host will no longer respond to osquery requests such as a live query. In an upcoming entry, we will explore how we might go about redeploying the `fleet` agent using MDM commands.


## BYOD enrollment

Bring Your Own Device enrollment is intended to allow users to enroll their personal devices into corporate management systems. While BYOD fosters user convenience and productivity, it introduces potential security vulnerabilities. Notably, the ability for users to unenroll their devices poses a significant challenge for sysadmins, as it compromises centralized device management and security measures.

Our examination of BYOD enrollment underscores the importance of vigilance and proactive measures. As sysadmins, it is crucial to verify that all devices are appropriately registered within the Apple Business Manager (ABM) account. Devices not present in ABM should be [manually added](https://support.apple.com/guide/apple-business-manager/add-devices-from-apple-configurator-axm200a54d59/web) to ensure comprehensive device oversight and security.


## Differences and considerations

The differences between ADE and BYOD enrollment methods are stark, each offering unique advantages and challenges for sysadmins. ADE prioritizes centralized control and security, making it the preferred choice for organizations seeking stringent device management protocols. In contrast, BYOD emphasizes user autonomy but necessitates heightened vigilance from sysadmins to mitigate security risks.

When evaluating enrollment methods, sysadmins must consider organizational security policies, user preferences, and device management capabilities. Zero-trust data access controls combined with a proactive approach to enrollment ensures that devices remain securely integrated into corporate ecosystems, minimizing potential vulnerabilities.


## Resolving enrollment discrepancies

Sysadmins must adopt a collaborative approach to addressing enrollment discrepancies, engaging with IT teams and end-users to standardize enrollment processes. Verifying device presence in ABM and implementing manual enrollment procedures where necessary are essential steps in bolstering device management strategies.

Looking ahead, sysadmins must remain vigilant in monitoring advancements in device management technologies. Staying informed and adaptable will be paramount in maintaining robust device security protocols.

We play a pivotal role in safeguarding organizational assets and data. The choice between ADE and BYOD enrollment methods underscores the balancing act between security and user autonomy. By understanding the nuances of each approach and implementing proactive measures, sysadmins can navigate device enrollment effectively, ensuring the integrity and security of corporate devices.





<meta name="articleTitle" value="Sysadmin diaries: device enrollment">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-05-03">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="In this sysadmin diary, we explore a the differences in device enrollment.">
