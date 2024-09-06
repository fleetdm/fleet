# Discovering Chrome AI using Fleet

![Discovering Chrome AI using Fleet](../website/assets/images/articles/discovering-chrome-ai-using-fleet-1600x900@2x.jpg)

# Discovering AI in Chrome with Fleet

Staying ahead of technological innovations is crucial for individuals and organizations. Google Chrome, one of the most widely used web browsers, continually evolves to incorporate new features, including artificial intelligence (AI). This article will guide you through detecting if AI capabilities have been enabled in Chrome using Fleet.

## Introduction to Chrome AI innovations

Google Chrome has integrated AI to enhance user experience by providing intelligent suggestions, improving search results, and offering in-browser assistance. Visit the [Chrome AI Innovations page](https://www.google.com/chrome/ai-innovations/) for more infomration.

## Using Fleet to detect AI features in Chrome

Fleet, a comprehensive device management and security tool, allows you to monitor various aspects of your devices, including software configurations and enabled features. Using Fleet, you can detect whether AI features are enabled in Chrome by querying device settings, specifically in the Chrome "Preferences" JSON file.

### Step 1: Understanding Chrome's preferences JSON file

Chrome stores user settings and configurations in a JSON file at the following path:

```
/Users/<user>/Library/Application Support/Google/Chrome/Default/Preferences
```

### Step 2: Identifying AI-related settings

AI-related features are stored in the `optimization_guide` section of the preferences. The `tab_organization_setting_state` field will tell you if AI-based tab management features are enabled:

`> jq` is a lightweight and powerful command-line tool for parsing, filtering, and manipulating JSON data. It allows you to extract specific information from JSON files efficiently. In this case, we use `jq` to locate and read the value of the `tab_organization_setting_state` key within Chrome's preference file which will help us understand how to craft our Fleet query for reporting the state of this setting.

- If enabled, the setting will return `1`.

![Chrome settings UI with Chrome AI enabled](../website/assets/images/articles/discovering-chrome-ai-using-fleet-1-1472x370@2x.png)

```
% jq '.optimization_guide.tab_organization_setting_state'  /Users/<user>/Library/Application\ Support/Google/Chrome/Default/Preferences                                      
1
```

- If disabled, the setting will return `2`.

![Chrome settings UI with Chrome AI disabled](../website/assets/images/articles/discovering-chrome-ai-using-fleet-2-1474x276@2x.png)

```
% jq '.optimization_guide.tab_organization_setting_state'  /Users/<user>/Library/Application\ Support/Google/Chrome/Default/Preferences                                      
2
```

### Step 3: Query the JSON file with Fleet

To query the JSON file and detect AI features using Fleet, you can use the following SQL query:

```
SELECT fullkey,path FROM parse_json WHERE path LIKE '/Users/%/Library/Application Support/Google/Chrome/Default/Preferences' AND fullkey='optimization_guide/tab_organization_setting_state';
```

### Conclusion

Following this guide, you've learned to detect whether AI features are enabled in Google Chrome using Fleet. Fleet's powerful querying abilities allow you to monitor these features across multiple devices, ensuring your organization's preferences and practices align.

<meta name="articleTitle" value="Discovering Chrome AI using Fleet">
<meta name="authorFullName" value="Brock Walters">
<meta name="authorGitHubUsername" value="nonpunctual">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-06">
<meta name="articleImageUrl" value="../website/assets/images/articles/discovering-chrome-ai-using-fleet-1600x900@2x.jpg">
<meta name="description" value="Use Fleet to detect and monitor settings enabled in Google Chrome by querying Chrome's preferences JSON file.">
