# Go-To-Market Architecture


## Automation


### LinkedIn comments from tracked posts

We track certian social posts from the [LinkedIn company page](https://www.linkedin.com/company/fleetdm/) using the following workflow:
- LinkedIn post URL provided to Clay.
- Clay enriches the data from any reactions or shares.
- Clay sends webhook to webhooks/receive-from-clay.js
- fleetdm.com sends a webhook to Salesforce.
- Salesforce will create/update the contact and account, and creates a "Historical event" for each contact.
- Clay then sends a webhook to Zapier.
- Zapier posts a message to the [_linkedin-comments-from-tracked-posts](https://fleetdm.slack.com/archives/C0AP1FM3ES2).


<img width="1410" height="1174" alt="image" src="https://github.com/user-attachments/assets/da2dccaa-e5ac-4373-9d93-d02b2a1bd8cd" />






<meta name="maintainedBy" value="sampfluger88">
<meta name="title" value="🚂 GTM architecture">
