# Frontend Testing Strategy & Plan

This document contains the testing strategy and plan for the fleet interface team. The testing strategy is a high level overview of the Who, What, When and Why when it comes to testing. The testing plan primarily outlines the How of testing and covers the different practices and toolings used in testing.

## Testing Strategy

### Testing Philosophy

When we create tests, we keep in mind how an end user will be using this software. This idea influences all other decisions when it comes to testing. Sometimes the end user is a company admin using the fleet UI, or a devops using fleetctl in the terminal, or a developer using a reusable UI component or utility or class; In any case we should first think about the end user when building our testing plan. Testing software front this perspective has many advantages including:

A focus on functionality and behaviour over implementation details. This leads to better maintainability of the testing suite which does not have to change as the implementation changes.
Gives a clear idea of what type of tests are useful and should be prioritised.
Gives higher confidence that the software behaves as intended in a real world scenario

### Who Tests

The developer is responsible for writing and maintaining tests as part of the work.

### What Types of Tests

We use a variety of testing to ensure that our software is working as intended. This includes:

 - End-to-end (e2e)
 - Integration
 - Unit
 - Static
 - Manual

#### Manual Testing

#### Static Analysis

This includes typing and linting to quickly ensure proper typings of data flowing through the application and that we are following coding conventions and styling rules. This give us a first of defense against writing buggy code.

#### Unit Testing

We unit test smaller reusable components that do not have many dependencies within them. These
components are primarily parametric based and require no or minimal mocking to be tested
effectively. They tend to be small building blocks of our application (e.g. reusable UI components,
common utilities, reusable hooks. The end user in this case tends to be other developers so we want
to ensure these components work as expected when used as building blocks.

#### Integration Testing

We use integration testing to strike a balance between speed and expense to write our tests. We use
them to test multiple components together to perform specific functionality. We also try to use
minimal mocking

#### E2E testing

These tests have a wide range when testing and resemble testing closest to how an end user of the
Fleet UI would use the application. Because of this, we test the important happy paths and errors
for user journeys and functionality at this level. We primarily do not mock services here and to
ensure our frontend, backend, and any other systems all work correctly together.

## Testing Plan

This section answers how we are testing our code. We cover tools and practices to writing tests.

### Tooling

This is the current tooling we are using to tests out code.

<img src="https://miro.medium.com/max/1400/1*iBBcTAf4zvn7yZq4K4MShA.png" width="400">
