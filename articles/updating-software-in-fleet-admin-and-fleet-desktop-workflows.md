# Updating software in Fleet: Admin and Fleet Desktop workflows

_Fleet Premium_

Fleet streamlines software updates with two parallel workflows: admins can update installed apps directly from the Fleet web UI, and end users can update select apps themselves from the Fleet Desktop Self-service section.

It’s important to note:
- **Admins can update any software on a host that displays an “Update available” status in the app, even if that software is not set as self-service.**
- **End users can update apps themselves from Fleet Desktop only if those apps are assigned as self-service.**

This guide covers the new software update experiences for both admins and end users.

## What’s new

When a newer version of software is uploaded to Fleet, devices with that software installed may be eligible for an update.

- If an app is set to self-service, end users see an **Updates** section in Fleet Desktop's **Self-service** tab and can update their own software with one click.
- If an app is not set to self-service, end users do not see the update in Fleet Desktop. **Admins can still update that software from the Fleet UI.**
- Real-time status and error information is displayed for both admins (in the Fleet UI) and end users (in Fleet Desktop, for self-service apps).

Learn more about the end user experience in the [Self-service guide](https://fleetdm.com/guides/software-self-service).

## Admin workflow: Updating software in the Fleet UI

Admins can review and update software for any host (and for any app where an available update is detected):

1. Go to **Hosts**.
2. Select a host.
3. Open the **Software** tab.
4. Open the **Library** tab.
5. In the **Library** table under the **status** column, look for the **Update available** to see software available for updates.  
   - This label appears for eligible updates, regardless of whether the app is set to self-service.
6. Select the indicator to open a modal with detailed version info, update progress, or failure details.
7. Trigger the update directly from this modal or from the **Actions** column.

This allows you to:
- Update software on managed devices, even for packages not enabled for end user self-service.
- Track user-initiated and admin-initiated updates in one place.
- Troubleshoot and confirm updates on the host.

## End user workflow: Updating via Fleet Desktop Self-service

End users can update their own apps using Fleet Desktop—**but only for apps set as self-service**:

- When a new version is published and self-service is enabled on that app, users see it in the new **Updates** section of the **Self-service** tab in **Fleet Desktop**.
- End users can update one or all available self-service apps with newer versions available through Fleet, and can view progress and error info directly.
- If an app isn't set to self-service, users will not see it in their Self-service section or be able to perform an update themselves (though admins still can from the web UI).

For more about enabling self-service, see the [Software self-service guide](https://fleetdm.com/guides/software-self-service).

> **Note:** If an app is installed in more than one location on a host, running the update may only upgrade one instance. The presence of outdated copies will continue to show as "Update available" until all outdated copies are updated or removed. This is a known limitation.

## How update eligibility works

Updates become available if:

- The app is assigned to the host’s team (or "No team").
- The device has the app installed and is detectable by Fleet.
- A newer app version is published to Fleet than at least one version of the app installed on the host.
- For VPP (App Store) apps, the device must have MDM enabled.
- Updates are not shown for software with missing version info or unsupported version comparison.

**Visibility:**
- **Admins:** See and can trigger any available update.
- **End users:** See and can trigger updates only for apps set as self-service.

For more technical detail and edge cases, refer to the [software self-service docs](https://fleetdm.com/guides/software-self-service).

## Update status and polling

Fleet provides real-time visibility into software update progress, no matter how the update is started. When an update is triggered—either by an admin in the Fleet UI or by an end user in Fleet Desktop—Fleet continuously monitors and surfaces the status for both workflows:

- **If the host is online:** The update status will poll automatically every 5 seconds to check for completion. Both admins and end users see real-time progress and results.
- **If the host is offline or unavailable:**  
  - Admins see an **"Update (pending)"** status in the Software Library table and in the host’s upcoming activity feed.
  - While pending, the update has not started on the host.
  - From the host detail page, admins may cancel a pending update from the **upcoming activity** feed if it has not yet started.

This ensures admins can track, manage, or cancel queued update actions, and both admins and end users always see the current update status in Fleet.

## Admin tips

- Use the **Software > Library** table to quickly identify and action pending updates.
- Use modal error details to troubleshoot failed updates reported by users.
- Use a Fleet policy and automatic install feature to help enforce version compliance.

## Additional resources

- [Self-service software guide](https://fleetdm.com/guides/software-self-service)
- [REST API: host software endpoint](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-software)
- [GitOps reference](https://fleetdm.com/docs/using-fleet/gitops#software)

## Conclusion

With Fleet, admins have full control to update any app when needed, while end users are empowered to update apps set as self-service. This dual approach streamlines patching, improves flexibility, and helps keep your organization’s apps secure and current.


<meta name="articleTitle" value="Updating software in Fleet: Admin and Fleet Desktop workflows">
<meta name="authorFullName" value="Rachel Perkins">
<meta name="authorGitHubUsername" value="rachelelysia">
<meta name="category" value="guides">
<!-- TODO: Confirm publish date -->
<meta name="publishedOn" value="2025-08-01">
<!-- TODO: Add image -->
<meta name="articleImageUrl" value="../website/assets/images/articles/building-an-effective-dashboard-with-fleet-rest-api-flask-and-plotly@2x.jpg">
<meta name="description" value="Learn how to update software as an admin via the Fleet UI or as an end user through Fleet Desktop.">
