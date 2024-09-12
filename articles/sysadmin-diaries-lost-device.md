# Sysadmin diaries: lost device

![Sysadmin diaries: lost device](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

Picture this: an employee calls you in a panic from an airport halfway across the country. They have just realized they left their company-issued laptop on the plane. Cue the sinking feeling. The device contains sensitive company data, and the thought of it falling into the wrong hands is enough to induce a cold sweat. But fear not! With Fleet's Mobile Device Management (MDM) capabilities, you can handle this situation swiftly and securely. Let us walk through how to lock or wipe a lost device using Fleet remotely.


## The scenario: a lost device

Imagine you receive a call from Jamie, a sales executive who has just landed in Chicago for a crucial client meeting. In their rush to deplane, they accidentally leave their laptop in the seatback pocket. Realizing the mistake after reaching the terminal, Jamie calls you, anxious and stressed about the potential data breach.


## Keep calm and use Fleet

First, take a deep breath. Fleet has got you covered using MDM. You can remotely lock and wipe the lost device to ensure your company’s data remains secure.


### Step 1: identify the device

Start by identifying the device in Fleet. Navigate to the **Hosts** page in the Fleet web UI. Use the search functionality to quickly find Jamie’s laptop by entering the hostname or any other relevant identifier.


### Step 2: remote lock


#### Using the Fleet web UI

1. Once you have located the device, click on it to open the **Host Overview** page.

2. In the **Actions** menu, select **Lock**.

3. A confirmation dialog will appear. Confirm that you want to lock the device.


#### Using the Fleet API

Alternatively, you can use the Fleet REST API to lock the device. Here is the API call you need to make:

``` bash

POST /api/v1/fleet/hosts/:id/lock

```

Replace `:id` with Jamie’s laptop's actual ID. This command sends a signal to lock the device as soon as it comes online. For macOS, this requires MDM to be enabled. For Windows and Linux, scripts need to be enabled.

If you wanted to call this from the command line, you could use `curl` with a command like this:

```bash

curl -X GET  https://fleet.company.com/api/v1/fleet/hosts/123/lock  -H "Authorization: Bearer <your_API_key>"

```


#### Optional steps for macOS

You can customize the locking message for macOS devices and set a PIN using an XML payload. Here is how:

1. Create a file named `command-lock-macos-host.xml` with the following content:

    ```xml

    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
        <key>Command</key>
        <dict>
            <key>Message</key>
            <string>This device has been locked. Contact IT on (123) 456-7890.</string>
            <key>PIN</key>
            <string>123456</string>
            <key>RequestType</key>
            <string>DeviceLock</string>
        </dict>
    </dict>
    </plist>

    ```

2. Customize the message and PIN as needed.

3. Safely store the recovery PIN using a secure method like 1Password.

4. Run the following command using the Fleet CLI tool, replacing `hostname` with the actual hostname in Fleet and the payload path with the file’s location:

    ```bash

    fleetctl mdm run-command --hosts=hostname --payload=command-lock-macos-host.xml

    ```


### Step 3: remote wipe (if necessary)

If you determine the device is at a high risk of being compromised, you may decide to wipe it. This is a more drastic step, but sometimes, it is necessary to protect sensitive information.


#### Using the Fleet web UI

1. On the same **Host Overview** page, go to the **Actions** menu and select **Wipe**.

2. Confirm the wipe action that appears in the dialog.


#### Using the Fleet API

To wipe the device via the API, use the following call:

```bash

POST /api/v1/fleet/hosts/:id/wipe

```

Again, replace `:id` with the device’s ID. The wipe command will be executed once the device is online. MDM must be enabled for macOS and Windows, and scripts must be enabled for Linux.


### Step 4: confirm and reassure

After you have locked and potentially wiped the device, inform Jamie of the steps actioned. Reassure them that the company’s data is now secure and provide any further instructions they may need, such as getting a replacement device.


### Unlocking macOS

If the device is found and needs to be unlocked:



1. Enter the security PIN (stored in Fleet, returned from the API call, or the XML file) in the device's input field.
2. The device will open to the regular login screen and ask for a password.
3. If the password is unavailable, select the option to enter the recovery key/disk encryption key (this option might be behind a ? icon).
4. Retrieve the disk encryption key from Fleet’s web UI.
5. Enter the disk encryption key on the laptop, which should prompt you to create a new password.
6. You will then be logged into the default device profile, which allows you to complete any needed actions (e.g., wiping or recovering data).


## Conclusion

Losing a device is stressful, but Fleet’s MDM capabilities can help you manage it effectively. You can protect sensitive data and prevent unauthorized access by remotely locking or wiping the lost device. Remember, stay calm, and rely on Fleet to secure your endpoints.

Fleet’s MDM features ensure that your data remains protected even if a device is lost. So, the next time you get that dreaded call, you will know exactly what to do.





<meta name="articleTitle" value="Sysadmin diaries: lost device">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-07-09">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="In this sysadmin diary, we explore what actions can be taken with Fleet when a device is lost.">
