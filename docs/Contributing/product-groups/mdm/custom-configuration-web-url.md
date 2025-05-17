## Custom configuration web URL

In Fleet, you can require end users to authenticate with your identity provider (IdP) before they can use their new Mac. Learn more [here](../Using%20Fleet/MDM-macOS-setup-experience.md#end-user-authentication-and-eula).

Some customers require end users to authenticate with a custom web application instead of an IdP.

How to require end users to authenticate with a custom web application:

1. Use Fleet's `team` YAML to create a "Workstations" team.

2. Create an automatic enrollment (DEP) profile w/ the `configuration_web_url` set to the URL of the custom web application and `await_device_configured` set to `true`.

3. In the "Workstations" `team` YAML, set the `macos_setup_assistant` option to the DEP profile.

4. In the Fleet UI, go to **Settings > Integrations > Automatic enrollment > Apple Business manager** and set the **Team** to "Workstations".

5. Update the custom web application to send a manual enrollment profile, with the end user's email, to a Mac after the end user enters valid credentials. Here's an example snippet of an enrollment profile:

```xml
<dict>
	<key>EndUserEmail</key>
	<string>user@example.com</string>
</dict>
```

You can use Fleet's API to [get the manual enrollment profile](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile).

6. Update the custom web application to wait until the fleetd agent is installed on the new Mac and then do the following steps.

7. Make a request to the [`GET /hosts` API endpoint](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) w/ the end user's email as a query param to get the Mac's hardware UUID. Example API request: `GET /hosts?query=user@example.com`.

8. Make a request to [Fleet's MDM command API](https://fleetdm.com/docs/rest-api/rest-api#run-custom-mdm-command) to pre-fill the end user's local macOS account via the [`AccountConfiguration` MDM command](https://developer.apple.com/documentation/devicemanagement/accountconfigurationcommand/command).

9. Make a request to Fleet's MDM command API to send the `Release Device from Await Configuration` MDM command to allow the device through to the next step in the set up.
