# Sysadmin diaries: passcode profiles

![Sysadmin diaries: passcode profiles](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

Passcode MDM profiles do not work the way we might think they should. We recently onboarded a new Fleetie, which is always an opportunity to _eat our own dog food_ when using Fleet for device management. 

This is the first in a series of things we encounter when managing our own devices (aka hosts). Fleet is an open-source and open-core company. Our [handbook](https://fleetdm.com/handbook) is public for everyone to view (and improve!). The [configuration policies](https://github.com/fleetdm/fleet-gitops) we apply to our devices reside in our public git repo. Today, we are looking at Fleet's password policy for macOS devices, which utilizes the [passcode policy payload](https://developer.apple.com/documentation/devicemanagement/passcode).

The user set up their new computer, created an account, and used a passcode that does not meet Fleet's passcode policy, namely a length of 10 characters. Sometime after that, the MDM profiles were delivered to the host. The expected behavior would be that the user would be prompted to enter a compliant passcode upon the next login.

What happens instead with this policy is that after login, the user is prompted with a "Password Policy Updated" notification.


![Password policy updated notification](../website/assets/images/articles/sysadmin-diaries-password-policy-updated-689x140@2x.png
"Password policy updated")


 This notification comes with the ability just to ignore it: Change Later or just dismiss the dialog.


![Password policy options > change now… or change
later](../website/assets/images/articles/sysadmin-diaries-change-later-231x160@2x.png "Password
policy > change later")


A quick search of the [Mac Admins Slack](https://www.macadmins.org/) confirmed my suspicions. The non-compliant passcode will remain indefinitely, and the profile requirements are only enforced on the next reset or new account creation.


### Why did this happen, and how do we solve it?

We discovered that the policy was not applied because Fleet needed to lock out account creation before all the policies had been successfully applied to the host. We have corrected this in [Fleet 4.48.0](https://fleetdm.com/releases/fleet-4.48.0), but how do we resolve this issue with an existing enrolled device and a change in the organization's password policy?


Do we add the `changeAtNextAuth` key? A read of Apple's documentation means every user with this policy must reset their password on the next authentication. That could be highly disruptive. And, if the policy is redeployed for any reason, could institute a password reset to every host in that team.


<blockquote purpose="large-quote">
<code>changeAtNextAuth</code> (boolean)

If true, the system causes a password reset to occur the next time the user tries to authenticate. If this key is set in a device profile, the setting takes effect for all users, and admin authentications may fail until the admin user password is also reset. Available in macOS 10.13 and later.
</blockquote>

Another solution is to use Fleet's remote script execution capability to trigger a one-off password reset on the host.

```
pwpolicy -u "501" -setpolicy "newPasswordRequired=1"
```

This will require the user to reset their password upon the next login to the host. This is likely the best solution in this situation, as it can be applied on an individual host basis.

In wrapping up this exploration into the intricacies of passcode profiles and their challenges, Fleet's open-source nature allows us to share these experiences and collectively seek solutions that enhance our understanding and implementation of device management policies. Let's continue the conversation. [Join us on Slack](https://fleetdm.com/support) and let us know how you might solve this issue and what device management problems you want to solve.




<meta name="articleTitle" value="Sysadmin diaries: passcode profiles">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-04-01">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="In this sysadmin diary, we explore a missapplied passcode policy.">
