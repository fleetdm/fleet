/**
 * Mocha's spec reporter, plus a per-profile-type tally printed after mocha's own epilogue.
 *
 * A reporter rather than an `after` hook: mocha prints its failure list once every hook has finished, so
 * anything a hook logged would end up buried above a run's worth of failing profiles.
 *
 * The run passes once every profile type reaches a 95% success rate.
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
      let typesBelowThreshold = [];
      let lines = [];
      for (let profileType of Object.keys(tallyByProfileType)) {
        let { passed, failed } = tallyByProfileType[profileType];
        let ran = passed + failed;
        totalRan += ran;
        totalPassed += passed;
        // The generator is not deterministic, so a run is judged on its success rate rather than on whether
        // a single generation out of hundreds came back wrong.
        if(passed / ran < 0.95) { typesBelowThreshold.push(profileType); }
        let label = PLATFORM_BY_PROFILE_TYPE[profileType] ? `${profileType} (${PLATFORM_BY_PROFILE_TYPE[profileType]})` : profileType;
        lines.push(`    ${label.padEnd(22)} ${String(ran).padStart(4)} ran   ${String(passed).padStart(4)} passed   ${percent(passed, ran).padStart(6)} success rate`);
      }
      this.passesSuccessThreshold = typesBelowThreshold.length === 0;

      console.log(
        '\n  ==== Profile generator summary ====\n\n' +
        (lines.length ? lines.join('\n') : '    (no generations ran)') + '\n' +
        `    ${'total'.padEnd(22)} ${String(totalRan).padStart(4)} ran   ${String(totalPassed).padStart(4)} passed   ${percent(totalPassed, totalRan).padStart(6)} success rate\n\n` +
        (this.passesSuccessThreshold ?
          '  Every profile type meets the 95% threshold.\n\n' :
          `  Below the 95% threshold: ${typesBelowThreshold.join(', ')}.\n\n`) +
        '  Note: Android is not implemented in the profile generator yet, Android profiles should be generated using Claude Code.\n'
      );
    });
  }

  // Mocha calls this, when defined, with the failure count it would otherwise exit with.
  done(failures, fn) {
    return fn(this.passesSuccessThreshold ? 0 : failures);
  }
}

module.exports = ProfileGeneratorSummaryReporter;
