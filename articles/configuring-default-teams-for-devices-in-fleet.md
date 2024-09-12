# Configuring default teams for macOS, iOS, and iPadOS devices in Fleet

Fleet allows you to configure default teams for macOS, iOS, and iPadOS devices as they automatically enroll in your instance. This ensures that devices are assigned to the correct teams and receive the appropriate apps and configuration profiles at enrollment.

## Why configure default teams?

The ability to assign default teams during device enrollment helps streamline the deployment process. Each device is automatically placed in its correct group, ensuring it receives the necessary configuration profiles and apps without requiring manual assignment.

### Configuring default teams in Fleet

Follow these steps to assign default teams to your devices:

1. **Navigate to automatic enrollment settings**:

   - Go to **Settings > Integrations > Mobile device management (MDM)**, and locate the **Automatic enrollment** section.

2. **Edit the ABM token**:

   - Click **Edit** next to the ABM token for which you want to configure default teams.

3. **Assign default teams**:

   - In the modal, use the dropdowns to select the appropriate default team for each platform (macOS, iOS, and iPadOS).

4. **Save your changes**: 

   - After selecting the teams, click **Save** to apply the changes. New devices will be automatically assigned to the selected teams upon enrollment.

## Benefits of configuring default teams

1. **Streamlined deployment**: Devices are configured and ready for use immediately after enrollment, reducing manual setup time.

2. **Reduced errors**: Automating team assignments helps avoid misconfigurations and ensures that the right profiles and apps are installed on the correct devices.

## Conclusion

Configuring default teams in Fleet simplifies the enrollment and management of Apple devices, ensuring that each device is assigned to the correct team immediately upon enrollment. This feature reduces manual setup tasks for IT teams by automating the assignment of configuration profiles and apps based on team specifications. By streamlining the deployment process and minimizing errors, configuring default teams ensures that devices are ready to use right out of the box, helping organizations save time and maintain consistency across their device fleet.

For organizations managing a large number of macOS, iOS, or iPadOS devices, this feature plays a crucial role in automating routine tasks, increasing efficiency, and improving the overall deployment experience. It enables teams to focus on more critical tasks and be confident that newly enrolled devices are correctly configured. For more information on using Fleet, please refer to the [Fleet documentation](https://fleetdm.com/docs) and [guides](https://fleetdm.com/guides).

<meta name="articleTitle" value="Configuring default teams for macOS, iOS, and iPadOS devices in Fleet">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-12">
<meta name="description" value="This guide will walk you through configuring default teams for devices using the Fleet web UI.">
