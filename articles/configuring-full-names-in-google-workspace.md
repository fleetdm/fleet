# Configuring full names in Google Workspace for Fleet integration

Fleet requires user full names to be configured in your Identity Provider (IdP) using specific attributes. Since Google Workspace doesn't natively provide a full name attribute that matches Fleet's requirements, this guide will walk you through setting up automatic synchronization of full names using Google's custom attributes and Apps Script.

## What we're solving

Fleet looks for full names in one of these attributes:
- `name`
- `displayname`
- `cn`
- `urn:oid:2.5.4.3`
- `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`

Google Workspace doesn't expose these attributes directly, so we need to:
1. Create a custom attribute to store full names
2. Write a script to automatically populate this attribute
3. Configure SAML mapping to use this attribute for Fleet

## Prerequisites

- Admin access to Google Workspace
- Basic understanding of Google Admin Console
- A Fleet instance configured for SAML authentication

## Step 1: Create a custom attribute in Google Workspace

First, we'll create a place to store the full name:

1. Sign in to the [Google Admin Console](https://admin.google.com)
2. Go to **Directory** > **Users**
3. Click the **More options** (three dots) in the top right
4. Select **Manage custom attributes**
5. Click **Add Custom Attribute**
6. Fill in the following:
   - **Category name**: `Fleet` (this should match the category name in the script)
   - **Description**: `Attributes for Fleet integration`
   - Under **Custom fields**, add a field:
     - **Name**: `Fullname`
     - **Info type**: `Text`
     - **Multi-value**: Leave unchecked
     - **Visibility**: `Visible to user and admin`
   - Click **Add**

## Step 2: Create a Google Apps Script to populate the attribute

Next, we'll create a script that automatically updates the full name attribute:

1. Go to [Google Apps Script](https://script.google.com)
2. Create a new project by clicking **New Project**
3. Delete any template code in the editor
4. Paste the following code:

```javascript
/**
 * Updates all users in Google Workspace with their full name in the Fleet.Fullname custom attribute.
 * This attribute can then be mapped in SAML configurations for Fleet integration.
 */
function updateFullnames() {
  const users = AdminDirectory.Users.list({customer: 'my_customer'}).users;
  
  users.forEach(user => {
    const fullName = `${user.name.givenName} ${user.name.familyName}`.trim();
    
    // Update custom schema using Fleet category and Fullname attribute
    const customSchemas = {
      Fleet: {
        Fullname: fullName
      }
    };
    
    try {
      AdminDirectory.Users.update(
        {customSchemas: customSchemas},
        user.primaryEmail
      );
      console.log(`Updated ${user.primaryEmail} with full name: ${fullName}`);
    } catch (error) {
      console.log(`Error updating ${user.primaryEmail}: ${error}`);
    }
  });
}

/**
 * Creates a daily trigger to run the updateFullnames function.
 * Run this function once to set up the automatic daily updates.
 */
function createTrigger() {
  // Check if trigger already exists
  const triggers = ScriptApp.getProjectTriggers();
  const triggerExists = triggers.some(trigger => 
    trigger.getHandlerFunction() === 'updateFullnames' && 
    trigger.getEventType() === ScriptApp.EventType.CLOCK
  );
  
  // Only create a new trigger if one doesn't already exist
  if (!triggerExists) {
    ScriptApp.newTrigger('updateFullnames')
      .timeBased()
      .everyDays(1)
      .create();
    console.log('Daily trigger created successfully');
  } else {
    console.log('Daily trigger already exists');
  }
}
```

5. Save the project by clicking **File** > **Save** (give it a name like "Fleet Full Names")
6. Click on the "+" icon next to "Services" in the left sidebar
7. Scroll down and select **Admin SDK API**
8. Click **Add**

## Step 3: Run the script and set up automation

Now we'll run the script and set up automatic updates:

1. In the Google Apps Script editor, select the `updateFullnames` function from the dropdown near the Run button
2. Click **Run**
3. You'll be prompted to authorize the script - follow the prompts:
   - Click "Review Permissions"
   - Select your Google account
   - Click "Allow" to grant the necessary permissions
4. The script will run and update all users' full names
5. Next, select the `createTrigger` function from the dropdown
6. Click **Run** again
7. This will set up a daily trigger to keep full names synchronized

## Step 4: Configure SAML Attribute Mapping

Finally, we'll map our custom attribute to one of Fleet's supported attributes:

1. Go back to the [Google Admin Console](https://admin.google.com)
2. Navigate to **Apps** > **Web and mobile apps**
3. Find your Fleet SAML app (or add it if you haven't already)
4. Go to the **SAML Attribute Mapping** section
5. Click **ADD MAPPING**
6. Configure the mapping:
   - **Google Directory Attribute**: Under Custom Attributes, select `Fleet > Fullname`
   - **App Attribute**: Enter `name` (or one of the other supported attributes, like `displayname`)
7. Click **SAVE**

## Verification

To verify everything is working correctly:

1. In Google Admin Console, check a user to confirm their custom attribute has been populated:
   - Go to **Directory** > **Users**
   - Click on a user
   - Look for the **Fleet** section and check if **Fullname** is populated
2. Trigger a SAML login to Fleet to verify the attribute is being passed correctly
3. Check if the macOS local account Full Name is correctly populated during device enrollment

## Troubleshooting

If you encounter issues:

- **Script errors**: Check the Apps Script execution logs by clicking on **Executions** in the left sidebar
- **Missing attributes**: Verify the custom attribute name matches exactly in both the script and SAML mapping
- **SAML issues**: Use Chrome Developer Tools during login to inspect the SAML response
- **Permissions**: Ensure your account has sufficient admin rights

## Maintenance

The script will automatically run daily to keep full names updated. If you make changes to the script, you may need to reauthorize it.

To monitor or manage the script's execution:
- View logs by clicking on **Executions** in the left sidebar
- Manage triggers by clicking on **Triggers** in the left sidebar

When new users are added to Google Workspace, they will be included in the next daily update cycle.

## Conclusion

You've now successfully configured Google Workspace to provide full names to Fleet using a custom attribute and automated synchronization. This setup ensures that Fleet can automatically populate and the macOS local account name for all your users during the initial macOS setup experience.


<meta name="articleTitle" value="Configuring full names in Google Workspace">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-02-25">
<meta name="description" value="Populating full name during macOS setup experience from Google Workspace">
