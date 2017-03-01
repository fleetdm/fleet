*   Fix issue with Distributed Query Pack results full screen feature that broke the browser scrolling abilities

*   Fix bug in which host counts in the sidebar did not match up with displayed hosts.

## Kolide 1.0.1 (February 27, 2017) ##

*   Fix an issue that prevented users from replacing deleted labels with a new label of the same name.

*   Improve the reliability of IP and MAC address data in the host cards and table.

*   Add full screen support for distributed query results.

*   Enable users to double click on queries and packs in a table to see their details.

*   Reprompt for a password when a user attempts to change their email address.

*   Automatically decorate the status and result logs with the host's UUID and hostname.

*   Fix an issue where Kolide users on Safari were unable to delete queries or packs.

*   Improve platform detection accuracy.

    Previously Kolide was determing platform based on the OS of the system osquery
    was built on instead of the OS it was running on. Please note: Offline hosts
    may continue to report an erroneous platform until they check-in with Kolide.

*   Fix bugs where query links in the pack sidebar pointed to the wrong queries.

*   Improve MySQL compatibility with stricter configurations.

*   Allow users to edit the name and description of host labels.

*   Add basic table autocompletion when typing in the query composer.

*   Support MySQL client certificate authentication. More details can be found in the [Configuring the Kolide binary docs](https://docs.kolide.co/kolide/1.0.1/infrastructure/configuring-the-kolide-binary.html)

*   Improve security for user-initiated email address changes.

    This improvement ensures that only users who own an email address and are
    logged in as the user who initiated the change can confirm the new email.

    Previously it was possible for Administrators to also confirm these changes
    by clicking the confirmation link.

*   Fix an issue where the setup form rejects passwords with certain characters.

    This change resolves an issue where certain special characters like "."
    where rejected by the client-side JS that controls the setup form.

*   Automatically login the user once initial setup is completed.
