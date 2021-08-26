# Security best practices

- [Describe your secure coding practices](#describe-your-secure-coding-practices-including-code-reviews-use-of-staticdynamic-security-testing-tools-3rd-party-scansreviews)
- [SQL injection](#sql-injection)
- [Broken authentication](#broken-authentication--authentication-session-management-flaws-that-compromise-passwords-keys-session-tokens-etc)
  - [Passwords](#passwords)
  - [Authentication tokens](#authentication-tokens)
- [Sensitive data exposure](#sensitive-data-exposure--encryption-in-transit-at-rest-improperly-implemented-APIs)
- [Cross-site scripting](#cross-site-scripting--ensure-an-attacker-cant-execute-scripts-in-the-users-browser)
- [Components with known vulnerabilities](#components-with-known-vulnerabilities--prevent-the-use-of-libraries-frameworks-other-software-with-existing-vulnerabilities)

The Fleet community follows best practices when coding. Here are some of the ways we mitigate against the OWASP top 10 issues:

## Describe your secure coding practices, including code reviews, use of static/dynamic security testing tools, 3rd party scans/reviews.

Every piece of code that is merged into Fleet is reviewed by at least one other engineer before merging. We don't use any security-specific testing tools.

The server backend is built in Golang, which (besides for language-level vulnerabilities) eliminates buffer overflow and other memory related attacks.

We use standard library cryptography wherever possible, and all cryptography is using well-known standards.

## SQL injection

All queries are parameterized with MySQL placeholders, so MySQL itself guards against SQL injection and the Fleet code does not need to perform any escaping.

## Broken authentication – authentication, session management flaws that compromise passwords, keys, session tokens etc.

### Passwords

Fleet supports SAML auth which means that it can be configured such that it never sees passwords.

Passwords are never stored in plaintext in the database. We store a `bcrypt`ed hash of the password along with a randomly generated salt. The `bcrypt` iteration count and salt key size are admin-configurable.

### Authentication tokens

The size and expiration time of session tokens is admin-configurable. See [The documentation on session duration](../2-Deploying/2-Configuration.md#session_duration).

It is possible to revoke all session tokens for a user by forcing a password reset.

## Sensitive data exposure – encryption in transit, at rest, improperly implemented APIs.

By default, all traffic between user clients (such as the web browser and fleetctl) and the Fleet server is encrypted with TLS. By default, all traffic between osqueryd clients and the Fleet server is encrypted with TLS. Fleet does not by itself encrypt any data at rest (_however a user may separately configure encryption for the MySQL database and logs that Fleet writes_).

## Broken access controls – how restrictions on what authorized users are allowed to do/access are enforced.

Each session is associated with a viewer context that is used to determine the access granted to that user. Access controls can easily be applied as middleware in the routing table, so the access to a route is clearly defined in the same place where the route is attached to the server see [https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189](https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L114-L189).

## Cross-site scripting – ensure an attacker can’t execute scripts in the user’s browser

We render the frontend with React and benefit from built-in XSS protection in React's rendering. This is not sufficient to prevent all XSS, so we also follow additional best practices as discussed in [https://stackoverflow.com/a/51852579/491710](https://stackoverflow.com/a/51852579/491710).

## Components with known vulnerabilities – prevent the use of libraries, frameworks, other software with existing vulnerabilities.

We rely on Github's automated vulnerability checks, community news, and direct reports to discover vulnerabilities in our dependencies. We endeavor to fix these immediately and would almost always do so within a week of a report.
