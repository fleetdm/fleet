const StackTraceParser = require('stacktrace-parser');
const mocha = require('mocha');

const { color } = mocha.reporters.base;

// Output failures with a single line trace to the first line of the test file
// found in the stack trace. For many errors this will speed up iteration time
// on fixing tests. Based on the example in the Mocha docs
// https://github.com/mochajs/mocha/wiki/Third-party-reporters
function Reporter(runner) {
  mocha.reporters.Spec.call(this, runner);

  runner.on('fail', (test, err) => {
    const lines = StackTraceParser.parse(err.stack);
    const line = lines.find(l => l.file.includes('.tests.js'));
    if (line) {
      // Error 000 helps this be identified by the default Emacs error parser
      console.log(
        color(
          'fail',
          `${line.file}(${line.lineNumber},${line.column}): Error 000: ${err.message.split('\n')[0]}`,
        ),
      );
    }
  });
}

// Extends from the default "spec" reporter
mocha.utils.inherits(Reporter, mocha.reporters.Spec);

module.exports = Reporter;
