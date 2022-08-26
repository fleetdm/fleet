# Contributing

## Filing issues

### Report a security vulnerability

Sensitive security-related issues should be reported to
[fleetdm.com/contact](https://fleetdm.com/contact) before a public issue is made.

## Contributing to documentation

Fleet currently uses [fleetdm.com/docs](https://fleetdm.com/docs) as the central location for documentation.

### Markdown links

Due to the structure of the Fleet documentation and GitHub's unique markdown files, there are several practices we'd like to call out if your documentation changes include links to other locations within the Fleet docs. 

#### Relative links

When including a link to a different file in the Fleet documentation, please use relative links when possible. 

For example, let's say you're working on changes in the Contribution docs and you'd like to add a link to the REST API docs. The relative link would look something like `../Using-Fleet/REST-API.md`.

#### Special characters in anchor links

There are certain characters GitHub doesn't support in the use of anchor links in markdown files. The general rule we've found is to only use a-z or A-Z characters in anchor links. All other characters should be removed.

For example, consider the section title *How do I connect to the Mailhog simulated server?*. The valid GitHub anchor link for this section is #how-do-i-connect-to-the-mailhog-simulated-server. Notice the *?* character is removed.

<meta name="pageOrderInSection" value="1000">
