# Cypress Testing

Cypress tests are designed solely for end-to-end testing. If this is your first time developing or running end-to-end tests, [Fleet testing documentation](../docs/03-Contributing/02-Testing.md) includes git instructions for test preparation and running tests.

## Fleet Cypress directories

### Integration directory

Cypress tests the integration of [entire features](integration/all/app) of the app.

With the roll out of teams, Cypress tests the user interface of each role of a user on the Premium Tier ([Fleet Premium Documentation](integration/premium/README.md)) and Free Tier ([Fleet Free Documentation](integration/free/README.md)).

### Support directory

[Commands](support/commands.ts) that are shared across tests are located in the support directory.

## Opening Cypress locally

To open simply run:

`yarn cypress:open`

This will open up cypress locally and
allow you to view the current test suite, as well as start writing new tests.

## Building best practices

As much as possible, build from a user's perspective. Use `.within` cypress command as needed to scope a command within a specific element (e.g. table, nav).

As much as possible, assert that the code is only selecting 1 item or that the final assertion is the appropriate count.

### Prioritization of selecting elements

1. By **element tag** using elements (e.g. buttons), we can target text within. Confirm what the user is seeing with target text. If this is not specific enough, add on Role.
2. By **role** using default or explicitly assigned roles of elements. If this is not specific enough, add on element class.
3. By **element class** is least preferred as it does not follow a user's perspective. Occasionally this may be the only option. If that is the case, prioritize using the class name that specifies what the element is doing.

## Resources

- [Fleet testing documentation](../docs/03-Contributing/02-Testing.md)
- [Cypress documentation](https://docs.cypress.io/api/table-of-contents)
- [React testing-library query documentation](https://testing-library.com/docs/queries/about/)
