/**
 * Mocha's spec reporter, plus a per-profile-type tally printed after mocha's own epilogue.
 *
 * A reporter rather than an `after` hook: mocha prints its failure list once every hook has finished, so
 * anything a hook logged would end up buried above a run's worth of failing profiles.
 *
 * Only tests tagged with a `profileType` (see configuration-profile-generator.test.js) are counted.
 */
const Mocha = require('mocha');
const { EVENT_TEST_PASS, EVENT_TEST_FAIL, EVENT_RUN_END } = Mocha.Runner.constants;

const PLATFORM_BY_PROFILE_TYPE = {
  mobileconfig: 'Apple',
  ddm: 'Apple',
  csp: 'Windows',
};


class ProfileGeneratorSummaryReporter extends Mocha.reporters.Spec {

  constructor(runner, options) {
    super(runner, options);

    let tallyByProfileType = {};
    let tally = (test, outcome)=>{
      if(!test.profileType) { return; }
      tallyByProfileType[test.profileType] = tallyByProfileType[test.profileType] || { passed: 0, failed: 0 };
      tallyByProfileType[test.profileType][outcome]++;
    };
    runner.on(EVENT_TEST_PASS, (test)=>{ tally(test, 'passed'); });
    runner.on(EVENT_TEST_FAIL, (test)=>{ tally(test, 'failed'); });

    // Registered after super(), so this runs after the spec reporter's epilogue.
    runner.once(EVENT_RUN_END, ()=>{
      let percent = (passed, ran)=>{ return ran ? `${(100 * passed / ran).toFixed(1)}%` : 'n/a'; };
      let totalRan = 0;
      let totalPassed = 0;
      let lines = [];
      for (let profileType of Object.keys(tallyByProfileType)) {
        let { passed, failed } = tallyByProfileType[profileType];
        let ran = passed + failed;
        totalRan += ran;
        totalPassed += passed;
        let label = PLATFORM_BY_PROFILE_TYPE[profileType] ? `${profileType} (${PLATFORM_BY_PROFILE_TYPE[profileType]})` : profileType;
        lines.push(`    ${label.padEnd(22)} ${String(ran).padStart(4)} ran   ${String(passed).padStart(4)} passed   ${percent(passed, ran).padStart(6)} success rate`);
      }

      console.log(
        '\n  ==== Profile generator summary ====\n\n' +
        (lines.length ? lines.join('\n') : '    (no generations ran)') + '\n' +
        `    ${'total'.padEnd(22)} ${String(totalRan).padStart(4)} ran   ${String(totalPassed).padStart(4)} passed   ${percent(totalPassed, totalRan).padStart(6)} success rate\n\n` +
        '  Note: Android is not implemented in the profile generator yet, Android profiles should be generated using Claude Code.\n'
      );
    });
  }
}

module.exports = ProfileGeneratorSummaryReporter;
