# Deploy Fleet on Render

> **Archived.** While still usable, this guide has not been updated recently. See the [Deploy Fleet](https://fleetdm.com/docs/deploy/deploy-fleet) docs for supported deployment methods.

![Deploy Fleet on Render](../website/assets/images/articles/deploy-fleet-on-render-800x450@2x.png)

[Render](https://render.com/) is a cloud hosting service that makes it easy to get up and running fast, without the typical configuration headaches of larger enterprise hosting providers. Our Render blueprint offers a one-click deploy of Fleet in under five minutes, and provides a scalable cloud environment with a lower barrier to entry, making it a great place to get some experience with [Fleet](https://fleetdm.com/) and [osquery](https://osquery.io/).

With one click, our Render blueprint will provision a Fleet web service, a MySQL database, and a Redis in-memory data store. Each service requires Render's `standard` plan at a cost of $7/mo each, totaling $21/mo to host your Fleet instance. If you prefer to follow a video, you can [watch us demonstrating the Render deployment process](https://youtu.be/hly0tAOqveA).

## Deployment steps

1. Open our [Render blueprint on GitHub](https://github.com/fleetdm/fleet/tree/main/infrastructure/render) and click the "Deploy to Render" button.
2. Create or log in to your Render account with associated payment information. 
3. Give your version of the blueprint a unique name like `yourcompany-fleet`. 
4. Click "Apply" for Render to provisions your services, which should take less than five minutes. 
5. When the services are done provisioning, click "Dashboard" in the Render navigation, where you will see your three new services. 
6. Click on the "Fleet" service to reveal your Fleet URL. Click on the URL to open your Fleet instance, then proceed to [setup Fleet and enroll hosts](#setup-fleet-and-enroll-hosts).

### MySQL

Fleet uses MySQL as the relational database to organize host enrollment and other metadata that powers Fleet.

### Redis

Fleet uses Redis to ingest and queue the results of distributed queries, cache data, and perform other data operations.

### Fleet

The Fleet server and user interface are packaged into a Docker image and hosted on Docker hub. Each time you run your blueprint, the Fleet image your web service is running will be updated with the [latest stable release](https://hub.docker.com/r/fleetdm/fleet/tags?page=&page_size=&ordering=&name=latest).

## Setup Fleet and enroll hosts

The first time you access your Fleet instance, you will be prompted with a setup page where you can enter your name, email, and password. Run through those steps to reach the Fleet dashboard.

> Set a strong and unique password instead of the default password during the setup process. 

You’ll find the enroll-secret after clicking “Add hosts”. This is a special secret the host will need to register to your Fleet instance. Once you have the enroll-secret you can use `fleetctl` to generate Fleet's agent (fleetd), which makes installing and updating osquery super simple.

To install `fleetctl`, which is the command line interface (CLI) used to communicate between your computer and Fleet, you either run `npm install -g fleetctl` or [download fleetctl](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.3.0) from Github. Once it's installed try the following command (Docker require) on your terminal:

```sh
fleetctl package --type=msi --enroll-secret <secret> --fleet-url https://<your-unique-service-name>.onrender.com
```

This command creates an `msi` installer pointed at your Fleet instance.

Now we need some awesome queries to run against the hosts we enroll, check out the collection [here](https://github.com/fleetdm/fleet/tree/main/docs/01-Using-Fleet/standard-query-library).

To get them into Fleet we can use `fleetctl` again. Run the following on your terminal:

```sh
curl https://raw.githubusercontent.com/fleetdm/fleet/main/docs/01-Using-Fleet/standard-query-library/standard-query-library.yml -o standard-query-library.yaml
```

Now that we downloaded the standard query library, we’ll apply it using `fleetctl`. First we’ll configure `fleetctl` to use the instance we just built.

Try running:

```sh
fleetctl config set --address https://<your-unique-service-name>.onrender.com
```

Next, login with your credentials from when you set up the Fleet instance by running `fleetctl login`:

```sh
fleetctl login
Log in using the standard Fleet credentials.
Email: <enter user you just setup>
Password:
Fleet login successful and context configured!
```

Applying the query library is simple. Just run:

```sh
fleetctl apply -f standard-query-library.yaml
```

`fleetctl` makes configuring Fleet really easy, directly from your terminal. You can even create API credentials so you can script `fleetctl` commands, and really unlock the power of Fleet.

That’s it! We have successfully deployed and configured a Fleet instance! Render makes this process super easy, and you can even enable auto-scaling and let the app grow with your needs.


<meta name="articleTitle" value="Deploy Fleet on Render">
<meta name="authorGitHubUsername" value="edwardsb">
<meta name="authorFullName" value="Ben Edwards">
<meta name="publishedOn" value="2021-11-21">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-render-800x450@2x.png">
<meta name="description" value="Learn how to deploy Fleet on Render.">
