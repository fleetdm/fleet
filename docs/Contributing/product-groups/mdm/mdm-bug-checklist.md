MDM Bug FAQ/Checklist

This checklist provides a list of questions contributors will commonly ask when attempting to triage and reproduce MDM bugs. Answering as many questions as possible from this checklist when gathering information about new a bug helps speed up the process and helps to avoid losing the context and information

1. OS Version of affected hosts  
   1. Windows: Name, edition and version, e.g. Windows 11 Professional 25H2. Build number like 10.0.26200.7462 is appreciated if possible  
   2. Apple hosts: Version number including minor version e.g. macOS 26.2  
   3. Android: Major version e.g. Android 16  
2. Fleetd version  
   1. Is fleet desktop enabled?  
   2. Any customizations to installer package?  
3. Serial numbers or other identifiers(e.g. UUID, host ID) of affected hosts
   * This should be communicated via a separate, private channel, like Slack or a `fleetdm/confidential` issue and referenced but **never shared in the public `fleetdm/fleet` repository**
4. Hardware level/platform of affected hosts  
   1. E.g. iPad 10th generation, M4 Macbook Pro  
   2. For Windows and Android, Manufacturer and Model is preferred  
5. Enrollment method/type of affected host(s) and approximate timeframe hosts were initial enrolled:  
   1. macOS:  
      1. ADE/ABM enrollment \- MDM status: On (company-owned)
         * Host in ADE/ABM but had manual profile installed?  
      2. Manual(profile-based) enrollment \- MDM status: On (manual)  
      3. Account driven enrollment \- MDM status: On (personal)  
   2. Windows:  
      1. Agent-driven enrollment \- MDM status: On (manual)
      2. Autopilot enrollment  On (company-owned)
      3. Settings app enrollment \- MDM status: On (manual)
   3. Android:  
      1. Work profile enrollment(e.g. Enroll page link) \- MDM status: On (personal)
      2. QR code enrollment at initial setup screen \- MDM status: On (company-owned)
      3. Google account based enrollment  
6. Were hosts ever managed by another MDM solution? If yes:  
   1. How were the hosts migrated:  
      1. ABM-based MDM migration (Apple)
      2. Fleet end-user migration flow?(macOS only)  
      3. Fleet automated migration flow?(Windows)  
      4. Manual migration
      5. Something else
   2. Is the management software from that solution installed?  
   3. Has it been configured to stop managing or disown hosts?  
7. Is any sort of application level allowlisting or denylisting installed?(e.g. Santa, Jamf Pro, etc). If yes:  
   1. Are any applications shipped by the OS vendor blocked?  
   2. Are any fleet components blocked?  
8. Is there any other configuration management software installed on the host? Are there any MDM profiles that could conflict?  
9. Do hosts require going through any sort of proxy, VPN or other connectivity to access internet resources?  
   1. Are fleet’s server URLs and update URLs allowlisted?  
   2. Are Apple’s cloud services(including APNS, ABM) allowlisted?  
10. Is the issue isolated to a single host, a single class of hosts(e.g. Specific team, version, or specific region), or all hosts? Does the issue persist after a reboot?  
   1. If possible: Does the issue persist after a wipe?  
11. Are any detail queries disabled for the team in Fleet?  
12. If the issue is enrollment related:  
    1. Is end user authentication enabled on the team?  
    2. Do hosts have fleetd installed before MDM enrollment?  
13. For Apple related issues:  
    1. Is APNS certificate valid? Has the APNS certificate been renewed recently?  
    2. Are ABM credentials valid? Have they been renewed lately?