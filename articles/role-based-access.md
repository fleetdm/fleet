# Role-based access

Users have different abilities depending on the access level they have.

## Roles

### Admin

Users with the admin role receive all permissions.

### Maintainer

Maintainers can manage most entities in Fleet, like queries, policies, and labels.

Unlike admins, maintainers cannot edit higher level settings like application configuration, fleets or users.

### Technician

`Applies only to Fleet Premium`

Technicians have the ability to run scripts, view their results, and install/uninstall software.

### Observer

The observer role is a read-only role. It can access most entities in Fleet, like queries, policies, labels, application configuration, fleets, etc.

They can also run reports configured with the `observer_can_run` flag set to `true`.

### Observer+

`Applies only to Fleet Premium`

Observer+ is an observer with the added ability to run *any* report.

### GitOps

`Applies only to Fleet Premium`

GitOps is a modern approach to Continuous Deployment (CD) that uses Git as the single source of truth for declarative infrastructure and application configurations.
GitOps is an API-only and write-only role that can be used on CI/CD pipelines.

## User permissions

See the [REST API](/docs/rest-api/rest-api) documentation for specific actions available to each role.


## Fleet-level user permissions

`Applies only to Fleet Premium`

Users in Fleet either have global access or access to specific fleets.

Users with access to specific fleets only have access to data for their assigned fleet.

Global users can access data for all fleets. Check out [the user permissions
table](#user-permissions) above for global user permissions.

Users can be assigned to multiple fleets, and can have different roles for each fleet. For example, a user can be given access to the "💻 Workstations" fleet and assigned the "Observer" role. This same user can be given access to the "📱🔐 Personal mobile devices" fleet and assigned the "Maintainer" role.


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-10-31">
<meta name="articleTitle" value="Role-based access">
<meta name="description" value="Learn about the different roles and permissions in Fleet.">
