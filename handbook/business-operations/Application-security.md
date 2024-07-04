# Application security

- [Describe your secure coding practices (SDLC)](https://fleetdm.com/handbook/business-operations/application-security#describe-your-secure-coding-practices-including-code-reviews-use-of-static-dynamic-security-testing-tools-3-rd-party-scans-reviews)
- [SQL injection](https://fleetdm.com/handbook/business-operations/application-security#sql-injection)
- [Broken authentication](https://fleetdm.com/handbook/business-operations/application-security#broken-authentication-authentication-session-management-flaws-that-compromise-passwords-keys-session-tokens-etc)
  - [Passwords](https://fleetdm.com/handbook/business-operations/application-security#passwords)
  - [Authentication tokens](https://fleetdm.com/handbook/business-operations/application-security#authentication-tokens)
- [Sensitive data exposure](https://fleetdm.com/handbook/business-operations/application-security#sensitive-data-exposure-encryption-in-transit-at-rest-improperly-implemented-apis)
- [Cross-site scripting](https://fleetdm.com/handbook/business-operations/application-security#cross-site-scripting-ensure-an-attacker-cant-execute-scripts-in-the-users-browser)
- [Components with known vulnerabilities](https://fleetdm.com/handbook/business-operations/application-security#components-with-known-vulnerabilities-prevent-the-use-of-libraries-frameworks-other-software-with-existing-vulnerabilities)

The Fleet community follows best practices when coding. Here are some of the ways we mitigate against the OWASP top 10 issues:

### Describe your secure coding practices, including code reviews, use of static/dynamic security testing tools, 3rd party scans/reviews.

Code commits to Fleet go through a series of tests, including SAST (static application security
testing). We use a combination of tools, including [gosec](https://github.com/securego/gosec) and
[CodeQL](https://codeql.github.com/) for this purpose.

At least one other engineer reviews every piece of code before merging it to Fleet.
This is enforced via branch protection on the main branch.

The server backend is built in Golang, which (besides for language-level vulnerabilities) eliminates buffer overflow and other memory related attacks.

We use standard library cryptography wherever possible, and all cryptography is using well-known standards.

### SQL injection

All queries are parameterized with MySQL placeholders, so MySQL itself guards against SQL injection and the Fleet code does not need to perform any escaping.

### Broken authentication – authentication, session management flaws that compromise passwords, keys, session tokens etc.

#### Passwords

Fleet supports SAML auth which means that it can be configured such that it never sees passwords.

Passwords are never stored in plaintext in the database. We store a `bcrypt`ed hash of the password along with a randomly generated salt. The `bcrypt` iteration count and salt key size are admin-configurable.

#### Authentication tokens

The size and expiration time of session tokens is admin-configurable. See [The documentation on session duration](https://fleetdm.com/docs/deploying/configuration#session-duration).

It is possible to revoke all session tokens for a user by forcing a password reset.

### Sensitive data exposure – encryption in transit, at rest, improperly implemented APIs.

By default, all traffic between user clients (such as the web browser and fleetctl) and the Fleet server is encrypted with TLS. By default, all traffic between osqueryd clients and the Fleet server is encrypted with TLS. Fleet does not by itself encrypt any data at rest (_however a user may separately configure encryption for the MySQL database and logs that Fleet writes_).

### Broken access controls – how restrictions on what authorized users are allowed to do/access are enforced.

Each session is associated with a viewer context that is used to determine the access granted to that user. Access controls can easily be applied as middleware in the routing table, so the access to a route is clearly defined in the same place where the route is attached to the server see [https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189](https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189).

### Cross-site scripting – ensure an attacker can’t execute scripts in the user’s browser

We render the frontend with React and benefit from built-in XSS protection in React's rendering. This is not sufficient to prevent all XSS, so we also follow additional best practices as discussed in [https://stackoverflow.com/a/51852579/491710](https://stackoverflow.com/a/51852579/491710).

### Components with known vulnerabilities – prevent the use of libraries, frameworks, other software with existing vulnerabilities.

We rely on GitHub's automated vulnerability checks, community news, and direct reports to discover
vulnerabilities in our dependencies. We endeavor to fix these immediately and would almost always do
so within a week of a report.

Libraries are inventoried and monitored for vulnerabilities. Our process for fixing vulnerable
libraries and other vulnerabilities is available in our
[handbook](https://fleetdm.com/handbook/security#vulnerability-management). We use
[Dependabot](https://github.com/dependabot) to automatically open PRs to update vulnerable dependencies.



<meta name="description" value="Explore Fleet's application security practices, including secure coding, SQL injection prevention, authentication, data encryption, access controls, and more.">
<meta name="maintainedBy" value="hollidayn">
