/**
 * Score the configuration profile generator against every case in
 * profile-generator/configuration-profile-generator-cases.js -- one live generation per `it()`, and
 * every case run REPEATS times, because a prompt that passes a case two runs in three is not passing it.
 *
 * The prompt comes from api/helpers/get-configuration-profile-generator-configuration.js, the same
 * helper the public endpoint calls, so there is nothing here to keep in sync with production.
 *
 * Every test costs a real model call, which is why this is not part of `npm test`.  Run it
 * deliberately:
 *
 *   sails_custom__anthropicSecret='…' npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' BASE_MODEL=claude-sonnet-5 npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' LOG_ALL_GENERATIONS=1 npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' REPEATS=1 npm run test-profile-generator
 *
 * Every result prints its profile, passing ones included, so the `readByEye` properties no assertion
 * covers can be confirmed by eye and a green run can be trusted rather than taken on faith.  To repeat one
 * case rather than all of them, to validate the output with contour, or to keep a transcript that can be
 * diffed against an earlier run, use the script instead -- it runs these same cases:
 *
 *   sails run test-llm-generated-configuration-profile --caseId=ddm-defer-minor-updates --parallelTests=5
 */
const assert = require('assert');
const util = require('util');

const { TEST_CASES, checkExpectations } = require('../configuration-profile-generator-cases');


// Overridable because the interesting question is usually whether a cheaper model can still pass
// these, and the answer changes with every model release.  Same default as the script.
const BASE_MODEL = process.env.BASE_MODEL || 'claude-haiku-4-5';

const LOG_ALL_GENERATIONS = process.env.LOG_ALL_GENERATIONS;

// Three, because the generator is not deterministic and one green run says less than it looks like it
// does: a case that fails one run in three is a case that fails.  Overridable, since iterating on a
// single case does not need three of everything, and each repeat is a real model call.
const REPEATS = Number(process.env.REPEATS || 3);
assert(Number.isSafeInteger(REPEATS) && REPEATS > 0 && REPEATS < 11, 'REPEATS must be a positive integer between 1 and 10');

// A profile the admin has to wait half a minute for is a broken feature even when the XML is
// perfect, so latency is an assertion rather than a note.  Checked after the content assertions, so
// a case that is both wrong and slow reports the wrong part first.
const MAX_ELAPSED_MS = 10000;


describe('configuration profile generator', function() {

  // The whole point of MAX_ELAPSED_MS is to catch the slow ones, so the mocha timeout has to sit well
  // above the budget rather than enforce it.
  this.timeout(60000);
  this.slow(MAX_ELAPSED_MS);

  before(function() {
    // Without a secret every single case would fail the same unexplained way, which reads as a
    // broken prompt rather than an unconfigured machine.
    if(!sails.config.custom.anthropicSecret) {
      console.log('    (skipping: sails.config.custom.anthropicSecret is not set -- re-run with sails_custom__anthropicSecret=… to actually generate anything.)');
      this.skip();
    }
  });

  // Derived from the cases rather than hard-coded, so adding a case of some new profile type cannot
  // silently leave it unrun.  Native array methods rather than lodash: `_` is a Sails global, and it
  // does not exist yet when mocha loads this file to register the tests.
  let profileTypesInOrder = TEST_CASES.reduce((typesSoFar, someCase)=>{
    if(!typesSoFar.includes(someCase.profileType)) { typesSoFar.push(someCase.profileType); }
    return typesSoFar;
  }, []);

  for (let profileType of profileTypesInOrder) {

    describe(profileType, function() {

      for (let testCase of TEST_CASES.filter((someCase)=>{ return someCase.profileType === profileType; })) {

        describe(`${testCase.id}: ${testCase.instructions}`, function() {

          // The repeats of one case are launched together here rather than awaited one at a time in the
          // tests below, so three generations cost one generation's wall clock.  The cases themselves stay
          // sequential: at most REPEATS requests are ever in flight, where launching every case at once
          // would measure the rate limit rather than the prompt.
          //
          // Each test then waits on its own repeat, which means the duration mocha prints for it is the
          // wait rather than the generation -- the first repeat carries most of it and the rest resolve
          // behind it.  The budget below is checked against the elapsed time each generation measured for
          // itself, so it stays the model's latency either way.
          let repeats = [];
          before(function() {
            repeats = Array.from({length: REPEATS}, ()=>{ return generateOnce(testCase); });
          });

          for (let repeatIdx = 0; repeatIdx < REPEATS; repeatIdx++) {

            it(`#${repeatIdx + 1}`, async function() {

              // Never a rejected promise: a generation that throws resolves to its error instead, since a
              // repeat that blew up before its own test got to await it would otherwise take down the
              // whole run as an unhandled rejection.
              let {rawResult, unexpectedError, elapsedMs} = await repeats[repeatIdx];

              // Distinguished from an abstention on purpose: collapsing the two would report a
              // misconfigured anthropicSecret as a model that refused to answer.
              assert(!unexpectedError, `unexpected error: ${unexpectedError && unexpectedError.message}`);

              let expectations = testCase.expect || {};

              // Mirrors the action's own acceptance test: an abstention, or a response missing any
              // required key, is not a usable profile.
              let abstained = (
                rawResult.couldNotGenerateProfile ||
                !rawResult.configurationProfile ||
                !rawResult.profileFilename ||
                !rawResult.settingsEnforced
              );

              if(expectations.expectFailure) {
                assert(abstained, 'expected this request to be refused, but a profile was generated');
                return;
              }

              assert(!abstained, `abstained but a profile was expected -- reason given: ${JSON.stringify(rawResult.reasonWhyAProfileCouldNotBeGenerated || '(none)')}`);

              let generatedProfile = {
                profile: rawResult.configurationProfile,
                profileFilename: rawResult.profileFilename,
                deliveryNotes: rawResult.deliveryNotes,
                items: rawResult.settingsEnforced,
              };

              let checkFailures = checkExpectations(expectations, generatedProfile);
              assert.strictEqual(
                checkFailures.length, 0,
                // The profile goes in the message rather than to stdout: a failure you cannot see is a
                // failure you cannot act on, and mocha only shows what the assertion carried.
                `${testCase.canary ? 'CANARY -- ' : ''}${checkFailures.length} check(s) failed:\n` +
                checkFailures.map((checkFailure)=>{ return `  - ${checkFailure}`; }).join('\n') +
                (testCase.readByEye ? `\n\nNote on this case: ${testCase.readByEye}` : '') +
                `\n\nprofileFilename: ${generatedProfile.profileFilename}` +
                `\ndeliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}` +
                `\n\n${generatedProfile.profile}\n\n` +
                `settingsEnforced:\n${util.inspect(generatedProfile.items, { depth: 4, colors: false })}\n`
              );

              // Passing the assertions is not the same as being right -- the checks are substrings, and the
              // `readByEye` properties exist precisely because some things no assertion covers.  So print
              // what was generated.  After the assertion rather than before it, so each result is shown
              // exactly once: a failure carries the same material in the message above.
              if(LOG_ALL_GENERATIONS) {
                console.log(
                  `\n      ---- ${testCase.id}${REPEATS > 1 ? ` #${repeatIdx + 1}` : ''}${testCase.canary ? ' (CANARY)' : ''} -- ${elapsedMs}ms ----\n` +
                  (testCase.readByEye ? `      CONFIRM BY EYE: ${testCase.readByEye}\n` : '') +
                  `      profileFilename: ${generatedProfile.profileFilename}\n` +
                  `      deliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}\n\n` +
                  `${generatedProfile.profile}\n\n` +
                  `      settingsEnforced:\n${util.inspect(generatedProfile.items, { depth: 4, colors: false })}\n`
                );
              }
              assert(elapsedMs <= MAX_ELAPSED_MS, `generated a passing profile, but took ${elapsedMs}ms, over the ${MAX_ELAPSED_MS}ms budget`);

            });
          }
        });
      }
    });
  }
});


/**
 * Generate one profile for one case, and time it.
 *
 * Resolves to what happened rather than rejecting, because these are launched together and awaited one at
 * a time: a repeat that failed before its own test reached it would otherwise surface as an unhandled
 * rejection and take the run with it.
 *
 * @param  {Dictionary} testCase
 * @returns {Dictionary}  {rawResult, elapsedMs} or {unexpectedError, elapsedMs}
 */
async function generateOnce(testCase) {
  let startedAt = Date.now();
  try {
    // The helper supplies the PayloadUUIDs and is called once per generation, so every repeat of a case
    // gets its own set.
    let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
      profileType: testCase.profileType,
      naturalLanguageInstructions: testCase.instructions,
    });
    let rawResult = await sails.helpers.ai.prompt.with({
      systemPrompt: generatorConfiguration.systemPrompt,
      prompt: generatorConfiguration.userPrompt,
      baseModel: BASE_MODEL,
      expectJson: true,
    });
    return { rawResult, elapsedMs: Date.now() - startedAt };
  } catch (err) {
    return { unexpectedError: err, elapsedMs: Date.now() - startedAt };
  }
}
