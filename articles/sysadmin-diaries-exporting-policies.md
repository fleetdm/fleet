# Sysadmin diaries: exporting policies

![Sysadmin diaries: exporting policies](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

As we explore using GitOps for managing Fleet, the need to streamline the transition of policies from the web user interface to the organization's GitOps configuration. In this latest diary entry, we will explore extracting policies initially created and tested in the web UI and implementing them in GitOps configurations. By leveraging existing tools and anticipating new features, we will explore both current methods and future capabilities to facilitate this essential task.


### Policy management in Fleet

Fleet provides a robust IT security and device management platform, allowing administrators to implement and manage policies across diverse operating systems. Integrating these policies with GitOps configurations is essential for maintaining consistency and automating policy enforcement in large-scale environments. Understanding the tools and methods available for this integration is the first step in optimizing your workflow.


### Extracting policies

Administrators have a couple of options to extract policies from the Fleet web UI for use in GitOps configurations.


#### Using the Fleet API

The most direct method is to use the Fleet API. You can extract existing policies by making a GET request to the Fleet server:

```
GET https://my.fleet.server/api/v1/fleet/global/policies
```

This request retrieves all global policies configured in the Fleet. The output, typically in JSON format, can then be converted into YAML format and integrated into your GitOps configurations. This process requires careful handling to ensure that the policy attributes are correctly mapped in the YAML file.


#### Manual integration process

For those preferring a hands-on approach, manually editing the `xxx.policies.yml` file is an alternative. This method involves:

- Navigate to the Fleet web UI and select the policy you wish to export.

- Copy and paste the relevant keys and values into your GitOps configuration file.

- Ensuring that features like `calendar_events_enabled` are `true` if the policy includes calendar events.


### Commands available

While `fleetctl` currently lacks a direct command to export policies in YAML format (`fleetctl get policies --yaml`), there are workarounds:

```
fleetctl api api/v1/fleet/teams/9/policies | jq .policies | yq -P
```

This command sequence uses `fleetctl` to call the API, `jq` to parse the JSON output, and `yq` to convert it into pretty YAML format. Although not as straightforward as a single command, this method is useful for teams needing to automate their workflow until a dedicated command is available.


### Upcoming Features and Improvements

Looking ahead, Fleet is committed to enhancing its GitOps integration capabilities. Upcoming features include:



* **Direct `fleetctl` commands for policy management:** Anticipated updates will introduce new `fleetctl` commands for easier policy export.
* **Enhanced API endpoints:** Future API enhancements will provide more granular control over policy attributes directly from the command line.


### Conclusion

Integrating policies from Fleet's web UI to GitOps configurations doesn't have to be complex. Administrators can streamline their workflows by using the current API methods and preparing for upcoming improvements, ensuring their environments remain secure and compliant. As Fleet continues to evolve, look forward to even more powerful tools that make policy management a seamless part of your IT operations.




<meta name="articleTitle" value="Sysadmin diaries: exporting policies">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-06-28">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="In this sysadmin diary, we explore extracting existing policies to enable gitops.">
