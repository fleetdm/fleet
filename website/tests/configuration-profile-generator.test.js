/**
 * Score the configuration profile generator against every case in
 * test/lib/configuration-profile-generator-cases.js -- one live generation per `it()`.
 *
 * The prompt comes from api/helpers/get-configuration-profile-generator-configuration.js, the same
 * helper the public endpoint calls, so there is nothing here to keep in sync with production.
 *
 * Every test costs a real model call, which is why this is not part of `npm test`.  Run it
 * deliberately:
 *
 *   sails_custom__anthropicSecret='…' npm run test-configuration-profile-generator
 *   sails_custom__anthropicSecret='…' npm run test-configuration-profile-generator
 *   sails_custom__anthropicSecret='…' BASE_MODEL=claude-sonnet-5 npm run test-configuration-profile-generator
 *
 * Every result prints its profile, passing ones included, so the `readByEye` properties no assertion
 * covers can be confirmed by eye and a green run can be trusted rather than taken on faith.  To find
 * out how OFTEN a case passes rather than whether it passed once, or to validate the output with
 * contour, use the script instead -- it runs these same cases and saves a transcript:
 *
 *   sails run test-llm-generated-configuration-profile --caseId=ddm-defer-minor-updates --parallelTests=5
 */
const assert = require('assert');
const util = require('util');

const { TEST_CASES, checkExpectations } = require('./lib/configuration-profile-generator-cases');


// Overridable because the interesting question is usually whether a cheaper model can still pass
// these, and the answer changes with every model release.  Same default as the script.
const BASE_MODEL = process.env.BASE_MODEL || 'claude-haiku-4-5';

let logOutput = process.env.LOG_ALL_GENERATIONS;
// A profile the admin has to wait half a minute for is a broken feature even when the XML is
// perfect, so latency is an assertion rather than a note.  Checked after the content assertions, so
// a case that is both wrong and slow reports the wrong part first.
const MAX_ELAPSED_MS = 10000;


describe('configuration profile generator', function () {

  // One live generation per test, and the whole point of MAX_ELAPSED_MS is to catch the slow ones,
  // so the mocha timeout has to sit well above the budget rather than enforce it.
  this.timeout(60000);
  this.slow(MAX_ELAPSED_MS);

  before(function () {
    // Without a secret every single case would fail the same unexplained way, which reads as a
    // broken prompt rather than an unconfigured machine.
    if (!sails.config.custom.anthropicSecret) {
      console.log('    (skipping: sails.config.custom.anthropicSecret is not set -- re-run with sails_custom__anthropicSecret=… to actually generate anything.)');
      this.skip();
    }
  });

  // Derived from the cases rather than hard-coded, so adding a case of some new profile type cannot
  // silently leave it unrun.  Native array methods rather than lodash: `_` is a Sails global, and it
  // does not exist yet when mocha loads this file to register the tests.
  let profileTypesInOrder = TEST_CASES.reduce((typesSoFar, someCase)=>{
    if (!typesSoFar.includes(someCase.profileType)) { typesSoFar.push(someCase.profileType); }
    return typesSoFar;
  }, []);

  for (let profileType of profileTypesInOrder) {

    describe(profileType, function () {

      for (let testCase of TEST_CASES.filter((someCase)=>{ return someCase.profileType === profileType; })) {

        it(`${testCase.id}: ${testCase.instructions}`, async function () {

          let startedAt = Date.now();
          // The helper supplies the PayloadUUIDs and is called once per test, so each generation
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
          let elapsedMs = Date.now() - startedAt;

          let expectations = testCase.expect || {};

          // Mirrors the action's own acceptance test: an abstention, or a response missing any
          // required key, is not a usable profile.
          let abstained = (
            rawResult.couldNotGenerateProfile ||
            !rawResult.configurationProfile ||
            !rawResult.profileFilename ||
            !rawResult.settingsEnforced
          );

          if (expectations.expectFailure) {
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
          if(logOutput){
            console.log(
              `\n      ---- ${testCase.id}${testCase.canary ? ' (CANARY)' : ''} -- ${elapsedMs}ms ----\n` +
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
