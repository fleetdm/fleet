# Discovering Geacon using Fleet

![Discovering Geacon using Fleet](../website/assets/images/articles/discovering-geacon-using-fleet-1600x900@2x.jpg)

A Go implementation of Cobalt Strike called 'Geacon' has been developed, which could make it easier for threat actors to target macOS devices. Enterprise security teams should be aware of the threat and take steps to protect their macOS devices. We will use Fleet to identify and locate Geacon payloads.


## Background

The Geacon project first appeared on GitHub four years ago, but it was not until recently that it was observed being deployed against macOS targets. Security researchers at [SentinelOne](https://www.sentinelone.com/blog/geacon-brings-cobalt-strike-capabilities-to-macos-threat-actors/) reported the activity this week after analysis of the payloads observed on VirusTotal suggested that two Geacon forks developed by an anonymous Chinese developer are responsible for the increase in Geacon activity. The forks are more popular than the original Geacon project, and they are being used to deliver malicious payloads to macOS devices. 


## Indicators of compromise

From SentinelOne's research, we are provided the indicators of compromise that can be used to discover Geacon on a device. We will use these indicators with osquery in Fleet to create a policy.


### Geacon SHA1s

6831d9d76ca6d94c6f1d426c1f4de66230f46c4a

752ac32f305822b7e8e67b74563b3f3b09936f89

bef71ef5a454ce8b4f0cf9edab45293040fc3377

c5c1598882b661ab3c2c8dc5d254fa869dadfd2a

e7ff9e82e207a95d16916f99902008c7e13c049d

fa9b04bdc97ffe55ae84e5c47e525c295fca1241


### Observed Geacon C2s

`47.92.123.17`

`13.230.229.15`


### BundleIdentifiers

`com.apple.ScriptEditor.id.1223`

`com.apple.automator.makabaka`


### Suspicious File Paths

`~/runoob.log`


### Building a query

Based on the indicators of compromise listed above, we have two ways to look for Geacon on a device. The first is a suspicious file path, and the second is BundleIdentifiers. Let us build a query for each.



1. In the top navigation of the Fleet UI, select **Queries**.
2. Select **Create new query** to navigate to the query console.
3. In the **Query** field, we will enter our query to first look for the suspicious file path. From above, we are looking for `runoob.log` in the user's directory, `~`. To do this, we will use the file and hash tables joined on the path, and look for our file path. \
`SELECT f.path, h.sha256 FROM file f JOIN hash h ON f.path = h.path WHERE f.path LIKE '/Users/%/runoob.log';`
4. Select **Save**, enter a name and description for your query, and select **Save query**.
5. Next, we will repeat this process to search for the suspicious BundleIdentifiers. To do this, we will look at the apps table and search for the BundleIdentifiers. \
`SELECT * FROM apps WHERE bundle_identifier = "com.apple.ScriptEditor.id.1223" or bundle_identifier = "com.apple.automator.makabaka"`


## Creating a policy

Now let us combine these two queries into a single policy.

1. In the top navigation of the Fleet UI, select Policies.
2. Select **Create new policy** to navigate to the query console.
3. In the **Query** field, enter our query that now combines both queries we built above into a single query. If either of these produces a match, the query will return `1` or true. \
`SELECT 1 WHERE EXISTS (SELECT 1 FROM file f JOIN hash h ON f.path = h.path WHERE f.path LIKE "/Users/%/runoob.log") OR (SELECT 1 FROM running_apps WHERE bundle_identifier = "com.apple.ScriptEditor.id.1223" or bundle_identifier = "com.apple.automator.makabaka") LIMIT 1;`
4. Select **Save**, enter a name and description for your policy, and select Save query.


## Discovering out-of-policy devices

The policies page indicates whether devices on each team are passing or failing with distinct "yes" or "no" responses. Although manually checking devices is relatively easy, we have made it easier for endpoint detection and response security using Fleet's automation.

Fleet can call a webhook when an out-of-policy device is identified. Users can specify a webhook URL to send alerts that include all devices that answered "No" to a policy. This makes it easier to create a support ticket and resolve each device.


## Conclusion

The Geacon project is a new threat to macOS devices. Enterprise security teams should be aware of the threat and take steps to protect their macOS devices. We have provided a detailed guide on how to identify and locate Geacon payloads using Fleet. We hope this information is helpful to you.

Here are some additional tips for protecting your macOS devices from Geacon and other threats:

* Add a [firewall rule](#using-packet-filter-on-macos-to-block-an-ip-address) blocking access to the IP addresses detailed in the indicators of compromise.
* Keep your macOS devices up to date with the latest security patches.
* Use a strong password manager to create and store strong passwords for all of your accounts.
* Be careful about what websites you visit and what files you open.
* Use a firewall to block unauthorized access to your macOS devices.

By following these tips, you can help to protect your macOS devices from Geacon and other threats.


## Using packet filter on macOS to block an IP address

Create a file that contains your firewall rules. You could name it anything, but for this example, let's call it `pf.rules`. 

```bash
sudo nano /etc/pf.rules
```

### Enter your rules

In the `pf.rules` file, write your rules as follows:

```bash
block drop out from any to 47.92.123.17
block drop out from any to 13.230.229.15
```

Save and exit the file (`Ctrl+X`, `Y`, then `Enter` for nano).

### Tell pfctl to load the rules

You then need to modify `pf.conf`, the main config file for `pfctl`, to load the rules from the `pf.rules` file. 

```bash
sudo nano /etc/pf.conf
```

Add the following line at the end of the file:

```bash
load anchor "customrules" from "/etc/pf.rules"
```

Then save and exit the file.

### Enable the Packet Filter

Finally, you need to reload the Packet Filter to apply the new rules:

```bash
sudo pfctl -e -f /etc/pf.conf
```

This will enable the firewall and load your rules.

### Verify the rules

To ensure your rules have been successfully loaded, use this command:

```bash
sudo pfctl -sr
```

You should see your rules listed.

> Please note that these steps require administrator access and that you should exercise caution when configuring firewall rules. Misconfigurations could block necessary network traffic or expose your system to security risks. It is a good practice to backup any configuration files before modifying them. You should back up your `/etc/pf.conf` file before editing it. 

These rules do not persist across reboots. If you want these rules to persist after rebooting your system, you should create a Launch Daemon to load the packet filter rules.


<meta name="articleTitle" value="Discovering Geacon using Fleet">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2023-05-18">
<meta name="articleImageUrl" value="../website/assets/images/articles/discovering-geacon-using-fleet-1600x900@2x.jpg">
<meta name="description" value="Enterprise security teams can use Fleet to identify and locate Geacon payloads and protect their macOS devices from this threat.">
