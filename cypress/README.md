# Cypress Testing

Cypress tests are designed solely for end-to-end testing. If this is your first time developing or running end-to-end tests, [Fleet testing documentation](https://github.com/fleetdm/fleet/blob/main/docs/4-Contribution/2-Testing.md) includes git instructions for test preparation and running tests.
## Opening Cypress locally

To open simply run:

`yarn cypress:open`

This will open up cypress locally and
allow you to view the current test suite, as well as start writing new test. 
## Bulding best practices

As much as possible, build from a user's perspective. Use .within cypress command as needed to scope a command within a specific element (e.g. <table>, <nav>).

As much as possible, assert that the code is only selecting 1 item or that the final assertion is the appropriate count.
### Prioritizization of selecting elements

1. By **element tag** using elements (e.g. buttons), we can target text within. Confirm what the user is seeing with target text. If this is not specific enough, add on Role.
2. By **role** using default or explicitly assigned roles of elements. If this is not specific enough, add on element class. 
3. By **element class** is least preferred as it does not follow a user's perspective. Occassionally this may be the only option. If that is the case, prioritize using the class name that specifies what the element is doing.  
## Resources

- [Fleet testing documentation](https://github.com/fleetdm/fleet/blob/main/docs/4-Contribution/2-Testing.md)
- [Cypress documentation](https://docs.cypress.io/api/table-of-contents)
- [React testing-library query documentation](https://testing-library.com/docs/queries/about)
