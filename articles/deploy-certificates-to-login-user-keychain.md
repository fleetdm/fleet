# Deploying certificates to the login(user) keychain on macos hosts

## Deploying the certificates

<!-- TODO -->

## Viewing local(user) keychain certificates on your hosts

You can now see whether certificates are installed in the system or user keychain—and who the user
is.

## Why it matters

Understanding where a certificate is installed helps you better assess its purpose and scope.
For example, system keychain certificates are typically installed by device administrators, while user keychain certificates may be added by specific users or applications.

Now, you can quickly:

- Identify user-installed certs across your fleet.
- Trace certificates back to the user account that installed them.
- Make more informed security and compliance decisions.

## How to use

1. In the Fleet UI, go to the Host Details for a Host and view the **Certificates** section.
2. You’ll now see a **Keychain** column for each certificate.
   - If the certificate is stored in the system keychain, the column will display `System`.
   - If it’s stored in the user keychain, it will display `User`.
3. **Hover over the word “User”** to see the username associated with that certificate.

## Use cases

- Investigate certificate issues tied to specific user accounts
- Identify certificates that may require revocation or follow-up
- Verify that certificates are installed in the appropriate keychain
