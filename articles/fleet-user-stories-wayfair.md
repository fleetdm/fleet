# Fleet user stories

## Ahmed Elshaer — DFIR, Blue Team, SecOps @ Wayfair

![Two people talking about Fleet](../website/assets/images/articles/fleet-user-stories-wayfair-cover-800x450@2x.jpeg)

This week, I spoke with Ahmed Elshaer (DFIR, Blue Team, SecOps) about how Wayfair uses Fleet and osquery:

### How did you first get started using osquery?

We were looking for a tool that provided linux logging, and incident response capabilities. Osquery had most of the requirements like logging, ability to scope an incident, interrogate systems but it’s missing the response or the ability to do an action on the remote systems.

### Why are you using Fleet?

We have POC’d couple free options and Fleet was the highest engagement and continuous development although it may be missing some features.
How do your end users feel about Fleet?

We are using Fleet only in the remote query on scale, so we find Fleet is doing a good job in that area, and it’s easy to use for any new members.

### How are you dealing with alert fatigue and false positives from your SIEM?

We have lots of queries that generate logs, but the ones that go into alerts are verified queries that are intended to hunt malicious or suspicious activity. Those activities are known based on public threat reports, Mitre Attack, or internal Red Team exercise.

### How could Fleet be better?

Fleet is doing a pretty good job now by listening to users in different channels, and I’ve seen lots of my discussions and ideas come into 

Fleet very fast, after a release or two. What I hope to see in Fleet are the following:

- Dashboard for all Assets and Labels, what are the online, offline, and new hosts metrics and over time.
- Audit logs for all fleet actions shipped to a remote logging destination
- Ability to create a notebook, list of queries that you can run ad-hoc like normal queries, which makes the IR process easier if you want to run and gather lots of data at once.

<meta name="category" value="success stories">
<meta name="authorGitHubUsername" value="mike-j-thomas">
<meta name="authorFullName" value="Mike Thomas">
<meta name="publishedOn" value="2021-08-20">
<meta name="articleTitle" value="Fleet user stories — Wayfair">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-user-stories-wayfair-cover-800x450@2x.jpeg">