# Deploy Fleet on Render

Render is a cloud hosting service that makes it easy to get up and running fast, without the typical configuration headaches of larger enterprise hosting providers.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/hly0tAOqveA?rel=0" frameborder="0" allowfullscreen></iframe>
</div>

### Prerequisites

- A Render account with payment information.

>The Fleet Render Blueprint will provision a web service, a MySQL database, and a Redis in-memory data store. At current pricing this will total **$65/month**.


### Instructions

<div purpose="deploy-to-render-button">
    <a href="https://render.com/deploy?repo=https://github.com/fleetdm/fleet" id="render-button" no-icon>
        <img src="https://render.com/images/deploy-to-render-button.svg" alt="Deploy to Render">
    </a>
</div>

1. Click "Deploy to Render" to open the Fleet Blueprint on Render. Ensure that the Redis instance is manually set to the same region as your other resources. You will be prompted to create or log in to your Render account with associated payment information.

2. Give the Blueprint a unique name like `yourcompany-fleet`.

3. Click "**Deploy Blueprint.**" Render will provision your services, which should take less than five minutes. 

4. Click the "**Dashboard**" tab in Render when provisioning is complete to see your new services.

5. From the "**Fleet**" service, click on the Fleet URL to open your Fleet instance. Then follow the on-screen instructions to set up your Fleet account.

> **Add a license key:** After successful deployment, navigate to the environment variables section of the Render blueprint to manually add your license key. 

> Support for add/install software features is coming soon. Get [commmunity support](https://chat.osquery.io/c/fleet).

<meta name="articleTitle" value="Deploy Fleet on Render">
<meta name="authorGitHubUsername" value="edwardsb">
<meta name="authorFullName" value="Ben Edwards">
<meta name="publishedOn" value="2025-07-17">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-render-800x450@2x.png">
<meta name="description" value="Learn how to deploy Fleet on Render.">
