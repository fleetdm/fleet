import requireHacker from 'require-hacker';

const fakeComponentString = `
  require('react').createClass({
    render () {
      return null;
    }
  })
`;

requireHacker.hook('svg', () => `module.exports = ${fakeComponentString}`);

