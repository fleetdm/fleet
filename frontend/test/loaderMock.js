import requireHacker from "require-hacker";

const fakeComponentString = `
  React = require('react');

  class FakeComponent extends React.Component {
    render () {
      return null;
    }
  }
`;

requireHacker.hook("svg", () => `module.exports = ${fakeComponentString}`);
