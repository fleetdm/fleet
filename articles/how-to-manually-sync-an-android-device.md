# How to manually sync an Android device

For Android hosts, there's no **Refetch** button on the **Host details** page in Fleet because Android hosts sync data automatically when they change. 

Sometimes changes don't appear immediately due to Google rate limiting. For testing, if you have physical access to the Android host, you can sync manually to speed things up:

1. Go to your Work Profile in **Settings**:
   1. Google devices: select your name at the top of **Settings**, then select your Work Profile.
   2. Samsung devices: select **Google**, then select your Work Profile.
2. Scroll to the bottom and select **Device Policy**.
3. Select the three dots in the upper right corner, then select **Sync policies**.
   - The message at the top of the screen should change to "Synced now".

On some devices, you may need to enable developer options first.

1. Go to **Settings > About phone**, and select **Build number** seven times.
   - A message will display during this period: "You are now _x_ steps away from being a developer."
2. Enter your PIN when prompted.
   - After successfully authenticating, a message will appear: "You are now a developer!"

To disable Developer options:

1. Go to **Settings**:
   - Google devices: select **System > Developer options**.
   - Samsung devices: select **Developer options** at the bottom of Settings, under About phone.
2. At the top, select the toggle to turn off Developer options.

<meta name="articleTitle" value="How to manually sync an Android device">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-04-09">
<meta name="description" value="Learn how to manually sync an Android device.">
