# Managing Google Chrome on Windows with Fleet

Use configuration profiles to enforce consistent Chrome browser settings across your Windows devices. Each profile that configures Chrome policies must also include the Google Chrome ADMX file. Without it, the policies fail to apply or fail verification.

**Prerequisites:**

- Administrative access to Fleet.
- Windows devices enrolled in Fleet.
- Basic familiarity with XML syntax and Group Policy concepts.


**Resources:**
- An example configuration profile with the Google Chrome ADMX embedded and a Chrome policy configured is available in our [GitHub solutions folder](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/admx%20Google%20Chrome.xml) (may not include the latest version of the ADMX)
- An example configuration profile, with the Google Chrome ADMX embedded, for enrolling your browsers into Chrome Enterprise Core for a Cloud-managed Chrome browser is available in our [GitHub solutions folder](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/enroll%20Google%20Chrome%20to%20enterprise%20console.xml)

---

## Step 1: Download the Google Chrome ADMX files

1. Download the latest Google Chrome ADMX templates from the official source: [Download Chrome ADMX templates (zip file)](https://chromeenterprise.google/download/#chrome-browser-policies)
2. Extract the ZIP file and locate the `chrome.admx` file in the `windows\admx` folder.

---

## Step 2: Create a configuration profile with the ADMX embedded

Windows needs the Chrome ADMX file to understand the Chrome policies you configure. Put the ADMX file and the Chrome policies in the same configuration profile. If you split Chrome policies across multiple profiles, include the ADMX in each one. For more information, see [Creating Windows CSPs: Ingesting custom ADMX templates](https://fleetdm.com/guides/creating-windows-csps#ingesting-custom-admx-templates-admxinstall).

1. Create a new `.xml` file in your editor of choice, and use the following template:

```xml
<!-- Ingest the Google Chrome ADMX -->
<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Chrome/Policy/ChromeAdmxFile</LocURI>
    </Target>
    <Data><![CDATA[PASTE_CHROME_ADMX_HERE]]></Data>
  </Item>
</Replace>

<!-- Configure Chrome policies -->
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>&lt;enabled/&gt;&lt;data id=&quot;RelaunchNotification&quot; value=&quot;2&quot;/&gt;</Data>
  </Item>
</Replace>
```

2. Replace `PASTE_CHROME_ADMX_HERE` with the entire contents of `chrome.admx`. The file starts with `<?xml version="1.0" ?>`. Keep it on the same line as `<![CDATA[`, with no line break or spaces in between: `<![CDATA[<?xml version="1.0" ?>`. If there's a line break, the profile fails with `status 500`.
3. Replace the policy `<Replace>` block with the Chrome policies you want. See the examples below.

### Example: enrolling devices in to Chrome Enterprise cloud management

If you would like to enroll your Chrome browsers to control the settings from the [Google cloud management portal](https://chromeenterprise.google/products/chrome-enterprise-core/), add this block after the ADMX `<Replace>` block. Replace the x's with your key.

```xml
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/CloudManagementEnrollmentToken</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>&lt;enabled/&gt;&lt;data id=&quot;CloudManagementEnrollmentToken&quot; value=&quot;xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx&quot;/&gt;</Data>
  </Item>
</Replace>
```

### Example: configuring `RelaunchNotification` and `RelaunchNotificationPeriod`

However, if you would like to manage your Chrome configuration by code, you can configure any settings available from the ADMX file. Add this block after the ADMX `<Replace>` block.

```xml
<Replace>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
    <Data>&lt;enabled/&gt;&lt;data id=&quot;RelaunchNotification&quot; value=&quot;2&quot;/&gt;</Data>
  </Item>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotificationPeriod</LocURI>
    </Target>
    <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
    <Data>&lt;enabled/&gt;&lt;data id=&quot;RelaunchNotificationPeriod&quot; value=&quot;259200000&quot;/&gt;</Data>
  </Item>
</Replace>
```

**Key points:**

- `<Format>`: Use `int` for integer (REG_DWORD) values and `chr` for string/boolean values.
- `<LocURI>`: The OMA-URI path for the policy. Refer to the [Chrome Enterprise Policy List](https://chromeenterprise.google/policies/) for valid paths.
- `<Data>`: The policy value. For boolean policies, include `<enabled/>` followed by the `<data>` tag.
- `RelaunchNotificationPeriod` values are in milliseconds. The example value `259200000` equals 3 days.

---

## Step 3: Deploy and verify

1. In Fleet, navigate to **Controls > OS settings > Configuration profiles** and add your new configuration profile.
2. You can **Refetch** the devices to apply the configuration sooner.
3. Verify the policies:
  - Open `regedit` on a target device and navigate to: `Computer\HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Google\Chrome`
  - Confirm that the policies (e.g., `RelaunchNotification`, `RelaunchNotificationPeriod`) appear with the correct values.
  - Restart Chrome and test the behaviour (e.g., check if the relaunch notification appears as configured).

---

## Troubleshooting

| Issue | Possible cause | Solution |
| ----------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Error during verification** | ADMX file not ingested | Ensure the profile includes the ADMX `<Replace>` block before the policy blocks. |
| **ADMXInstall fails with `status 500`** | Line break or spaces between `<![CDATA[` and `<?xml ...?>` | Put `<?xml ...?>` on the same line as `<![CDATA[`, with nothing in between. |
| **Policies not applying** | Incorrect `<Format>` or `<LocURI>` | Double-check `<Format>` (e.g., `int` for REG_DWORD) and the OMA-URI path. |
| **ADMX file not found** | Incorrect `<LocURI>` in the ADMX `<Replace>` block | Verify the path in the `<Target>` section matches Fleet's expected location. |
| **Device sync failures** | Network or Fleet agent issues | Check the Fleet agent logs on the device for errors. |

---

## References

- [Google Chrome Enterprise Policy List](https://chromeenterprise.google/policies/)
- [Fleet documentation: Creating Windows CSPs](https://fleetdm.com/guides/creating-windows-csps)
- [Microsoft ADMX guide](https://learn.microsoft.com/en-us/troubleshoot/browsers/group-policy-admx)
- [Example solutions folder](https://github.com/fleetdm/fleet/tree/main/docs/solutions)

---

## Next steps

- Explore additional Chrome policies in the [Chrome Enterprise Policy List](https://chromeenterprise.google/policies/).
- Test policies in a staging environment before fleet-wide deployment.

<meta name="articleTitle" value="Managing Google Chrome on Windows with Fleet">
<meta name="authorFullName" value="Gray Williams">
<meta name="authorGitHubUsername" value="GrayW">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-05-04">
<meta name="description" value="Learn to manage Google Chrome on Windows with Fleet by deploying the Chrome ADMX file and configuring browser policies using configuration profiles.">
