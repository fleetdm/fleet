# Configuring an alternative Apple MDM URL

Fleet [v4.59.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.59.0) allows setting an alternative MDM URL helps organizations differentiate MDM traffic from other Fleet traffic, allowing the application of network rules specific to MDM communications. The `mdm.apple_server_url` configuration specifies the URL that Apple devices use to communicate with your Fleet instance for MDM purposes. This configuration is optional; if not set, MDM will default to using the Fleet Server URL.

However, be aware that changing this URL after devices have been enrolled in MDM requires those devices to be re-enrolled into MDM.

## Prerequisites

* Fleet v4.59.0

## Step-by-step instructions

1. Prepare your DNS

    Create a DNS record Fleet can use for Apple MDM traffic

    **Example:**
    * Fleet Server URL: `https://fleet.example.com 104.21.82.73`
    * Apple Server URL: `https://fleet-mdm.example.com 104.21.82.73`

    Both URLs should point to the same IP address to ensure seamless handling of both MDM and non-MDM traffic.

2. Configure the Apple server URL in Fleet

    Via the Fleet UI:

    * **Access Fleet UI**: Navigate to **Settings > Organization settings > Advanced options > Apple server URL**.
    * **Set the URL**: Enter your MDM Apple Server URL.
    * **Apply changes**: Run the following command to apply your changes:
  
    Via GitOps:

    ```yaml
    org_settings:
      mdm:
        apple_server_url: "https://mdm.example.com"
    ```

    See the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#policies) for an example.

## Conclusion

The Apple Server URL is an optional configuration that allows you to route MDM traffic through a separate URL, which can be beneficial for monitoring and controlling MDM traffic separately from other Fleet communications. 

**Important**: Be sure to set the Apple Server URL **before** enrolling devices to avoid the need for device MDM re-enrollment.

<meta name="articleTitle" value="Configuring an Alternative Apple MDM URL">
<meta name="authorFullName" value="Tim Lee">
<meta name="authorGitHubUsername" value="mostlikelee">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-11-01">
<meta name="description" value="A guide on configuring an alternative Apple MDM URL in Fleet for better traffic management.">
```
