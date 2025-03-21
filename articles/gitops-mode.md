# GitOps mode

The Fleet UI now supports GitOps mode, which helps users avoid unexpected changes when using GitOps
to configure Fleet by preventing users from manually updating
GitOps-configurable settings and features in the UI.

For example, if a user uses the UI to add
a new saved query and then runs GitOps, the manual change will be overwritten by the GitOps
configuration. To help avoid this potentially confusing situation, GitOps mode prevents the user
from manually saving or editing the query in the first place (though does still allow running an ad-hoc live query):

![](../website/assets/images/articles/gitops-mode-disables-saving-queries.png)

## Enabling GitOps mode
To turn GitOps mode on or off, navigate to **Settings** > **Integrations** > **Change management**:

![](../website/assets/images/articles/enabling-gitops-mode.gif)

## What it covers

GitOps mode prevents the UI user from editing [GitOps-configurable settings and
features](https://fleetdm.com/docs/configuration/yaml-files). They will still be able to, for example:
- Read any data presented in the UI
- Add and edit users
- Add and edit labels
- Run live queries

## More on GitOps
<!-- TODO - update to link to Allen's article! -->

- [Why use GitOps to configure Fleet?](https://www.example.com)
- [Preventing Mistakes with GitOps](https://fleetdm.com/guides/articles/preventing-mistakes-with-gitops)

<meta name="articleTitle" value="GitOps mode">
<meta name="authorFullName" value="Jacob Shandling">
<meta name="authorGitHubUsername" value="jacobshandling">
<meta name="publishedOn" value="2025-03-21">
<meta name="category" value="guides">
<meta name="description" value="Use GitOps mode to prevent UI users from updating GitOps-managed features">