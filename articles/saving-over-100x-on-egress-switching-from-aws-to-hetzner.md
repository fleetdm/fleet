# Saving over 100x on egress switching from AWS to Hetzner

![Deploying Fleet on AWS with Terraform](../website/assets/images/articles/saving-over-100x-on-egress-switching-from-aws-to-hetzner-cover-800x533@2x.jpeg)
*Egrets? No, egress.*

Our AWS CloudFront bill spiked to **$2,457** in October 2021 from **$370** in September. When we dug into the bill, we saw that egress in the EU region accounted for most of this increase, with egress in the US making up the rest.

This wasn’t an indication of some misconfiguration on our end, but rather, a symptom of success. Our primary product is [Fleet](https://fleetdm.com/), an [open core](https://github.com/fleetdm/fleet) platform for device management built on [osquery](https://osquery.io/). We offer an update server for agent updates that is freely accessible to both community users and our paying customers. Getting these costs under control became a priority so that we could continue to offer free access.

Our needs for this server are pretty simple. We generate and sign static metadata files with [The Update Framework](https://theupdateframework.io/), then serve those along with the binary artifacts. We don’t have any strict requirements around latency, as these are background processes being updated.

At first we looked at Cloudflare’s free tier; Free egress is pretty appealing. Digging into Cloudflare’s terms, we found that they only allow for free tier caching to be used on website assets. To avoid risking a production outage by violating these terms, we got in touch with them for a quote. This came out to about a **2x savings** over AWS. But we knew we needed orders of magnitude savings in order to expand our free offering.

Having heard of Hetzner’s low egress costs (20TB free + €1.19/TB/month), we investigated what it would take to run our own server. We stood up a [Caddy file server](https://caddyserver.com/docs/caddyfile/directives/file_server) with automatic HTTPS via Let’s Encrypt over the course of a few hours.

Our December Hetzner bill came out to **€36.75 ($41.63)**. This represents a savings of **59x** over our prior AWS bill, putting us solidly in the range to continue offering the free update server. We can still double our egress with Hetzner before incurring additional charges, which will render a savings of over **118x** from AWS. Beyond that, the additional egress costs should remain reasonable.

DIYing it does come with additional maintenance burden, but so far we’ve found this manageable. Caddy on Hetzner has proved exceptionally reliable, with well over 99% uptime in the last two months and no manual interventions required.

---

Fleet is building an open future for device management, starting with the [most widely deployed osquery fleet manager](https://fleetdm.com/).

Are you interested in working full-time in [Fleet’s public GitHub repository](https://github.com/fleetdm/fleet)? We’re [hiring remote engineers](https://fleetdm.com/jobs), worldwide.

<meta name="category" value="engineering">
<meta name="authorGitHubUsername" value="zwass">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="publishedOn" value="2022-01-25">
<meta name="articleTitle" value="Saving over 100x on egress switching from AWS to Hetzner">
<meta name="articleImageUrl" value="../website/assets/images/articles/saving-over-100x-on-egress-switching-from-aws-to-hetzner-cover-1600x900@2x.jpg">