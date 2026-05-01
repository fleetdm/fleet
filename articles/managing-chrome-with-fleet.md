# Managing Google Chrome with Fleet

Use configuration profiles to enforce consistent Chrome browser settings across your Windows devices. Before configuring Chrome policies, you must first deploy the Google Chrome ADMX file to your devices — skipping this step will cause errors during verification or prevent policies from applying.

**Prerequisites:**

- Administrative access to Fleet.
- Windows devices enrolled in Fleet.
- Basic familiarity with XML syntax and Group Policy concepts.

---

## Step 1: Download the Google Chrome ADMX files

1. Download the latest Google Chrome ADMX templates from the official source: [Download Chrome ADMX templates (zip file)](https://chromeenterprise.google/download/#chrome-browser-policies)
2. Extract the ZIP file and locate the `chrome.admx` file in the `windows\admx` folder.

---

## Step 2: Deploy the ADMX file to the device

To apply Chrome policies, Windows needs the ADMX file to understand what settings are being configured. Do this by creating a configuration profile that deploys the file to your devices. For more information, see [Creating Windows CSPs: Ingesting custom ADMX templates](https://fleetdm.com/guides/creating-windows-csps#ingesting-custom-admx-templates-admxinstall).

### Create a configuration profile for ADMX ingestion

1. In Fleet, navigate to **Configuration profiles** and create a new profile.
2. Use the following XML to upload the `chrome.admx` file to the device's policy store:

```xml
<Add>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
      <Type>text/plain</Type>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Chrome/Policy/ChromeAdmxFile</LocURI>
    </Target>
    <Data><![CDATA[
      <!-- Paste the full contents of chrome.admx here -->
    ]]></Data>
  </Item>
</Add>
```

**Note:**

- Replace `<![CDATA[ ... ]]>` with the entire contents of the `chrome.admx` file.
- This ensures the ADMX file is available for policy configuration on target devices.

---

## Step 3: Configure Chrome policies

Once the ADMX file is deployed, configure Chrome policies using the `<Replace>` block in a new or existing configuration profile.

### Example: configuring `RelaunchNotification` and `RelaunchNotificationPeriod`

Use the following XML to enforce Chrome policies. Note the use of `int` for REG_DWORD values (e.g., `RelaunchNotificationPeriod`):

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

## Step 4: Deploy and verify

1. Assign the profile to the desired devices or groups in Fleet.
2. **Refetch** the devices to apply the configuration.
3. Verify the policies:
  - Open `regedit` on a target device and navigate to: `Computer\HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Google\Chrome`
  - Confirm that the policies (e.g., `RelaunchNotification`, `RelaunchNotificationPeriod`) appear with the correct values.
  - Restart Chrome and test the behaviour (e.g., check if the relaunch notification appears as configured).

---

## Troubleshooting

| Issue | Possible cause | Solution |
| ----------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Error during verification** | ADMX file not ingested | Ensure the profile was deployed successfully. |
| **Policies not applying** | Incorrect `<Format>` or `<LocURI>` | Double-check `<Format>` (e.g., `int` for REG_DWORD) and the OMA-URI path. |
| **ADMX file not found** | Incorrect `<LocURI>` in the `<Add>` block | Verify the path in the `<Target>` section matches Fleet's expected location. |
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
