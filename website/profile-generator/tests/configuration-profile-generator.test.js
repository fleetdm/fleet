/**
 * Score the configuration profile generator against every case in
 * profile-generator/configuration-profile-generator-cases.js -- one live generation per `it()`, and
 * every case run REPEATS times, because a prompt that passes a case two runs in three is not passing it.
 *
 * Each generation goes through api/helpers/generate-configuration-profile.js, the same helper the public
 * endpoint calls, so there is nothing here to keep in sync with production: the prompts, the model each
 * profile type is generated with, and the post-processing are all production's.
 *
 * Every case runs the way the profile generator page asks for a profile, with a profile type.  The Apple
 * cases also run the way Fleet asks, with a platform, which routes each one to a declaration or a
 * .mobileconfig first; whether it went where it should is tallied on its own.
 *
 * Every test costs a real model call, which is why this is not part of `npm test`.  Run it
 * deliberately:
 *
 *   sails_custom__anthropicSecret='…' npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' SUITE=page npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' SUITE=fleet npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' ROUTING_ONLY=1 npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' BASE_MODEL=claude-haiku-5-5 DISABLE_THINKING=1 npm run test-profile-generator
 *   sails_custom__anthropicSecret='…' BASE_MODEL=claude-sonnet-5-5 EFFORT=medium npm run test-profile-generator
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

const { TEST_CASES, checkExpectations, checkDeclineReason, fleetRequestFor, isWorthRetryingAfterAPause } = require('../configuration-profile-generator-cases');


// Overridable because the interesting question is usually whether a cheaper model can still pass
// these, and the answer changes with every model release.  Unset, each profile type is generated the
// way production generates it.
const BASE_MODEL = process.env.BASE_MODEL || undefined;
const EFFORT = process.env.EFFORT || undefined;
// Unset leaves each profile type's production setting alone; DISABLE_THINKING=0 turns thinking back on.
const DISABLE_THINKING = process.env.DISABLE_THINKING ? process.env.DISABLE_THINKING !== '0' : undefined;

const LOG_ALL_GENERATIONS = process.env.LOG_ALL_GENERATIONS;

// Three, because the generator is not deterministic and one green run says less than it looks like it
// does: a case that fails one run in three is a case that fails.  Overridable, since iterating on a
// single case does not need three of everything, and each repeat is a real model call.
const REPEATS = Number(process.env.REPEATS || 3);
assert(Number.isSafeInteger(REPEATS) && REPEATS > 0 && REPEATS < 11, 'REPEATS must be a positive integer between 1 and 10');

// SUITE=page runs every case as the profile generator page asks, and SUITE=fleet runs the Apple cases as
// Fleet asks; unset runs both.  ROUTING_ONLY=1 checks only where each Apple case is routed, which takes the
// router's call alone and none of the generations, for working on the router.
const SUITE = process.env.SUITE || undefined;
assert([undefined, 'page', 'fleet'].includes(SUITE), 'SUITE must be "page" or "fleet" when it is set');
const ROUTING_ONLY = !!process.env.ROUTING_ONLY;

// PROFILE_TYPE=csp runs only the cases written for one profile type, as Fleet asks for them too.
const PROFILE_TYPE = process.env.PROFILE_TYPE || undefined;
const CASES_TO_RUN = PROFILE_TYPE ? TEST_CASES.filter((someCase)=>{ return someCase.profileType === PROFILE_TYPE; }) : TEST_CASES;
assert(CASES_TO_RUN.length > 0, `No cases have profileType ${PROFILE_TYPE}`);

// WRITE_PROFILES_FROM_SETTINGS=0 has the model write each .mobileconfig and Windows profile itself, to compare against.
const WRITE_PROFILES_FROM_SETTINGS = process.env.WRITE_PROFILES_FROM_SETTINGS ? process.env.WRITE_PROFILES_FROM_SETTINGS !== '0' : undefined;

// A profile the admin has to wait half a minute for is a broken feature even when the XML is
// perfect, so latency is an assertion rather than a note.  Checked after the content assertions, so
// a case that is both wrong and slow reports the wrong part first.
const MAX_ELAPSED_MS = 10000;


if(!ROUTING_ONLY && SUITE !== 'fleet') {
  describe('configuration profile generator', function() {

    // The whole point of MAX_ELAPSED_MS is to catch the slow ones, so the mocha timeout has to sit well
    // above the budget rather than enforce it.
    this.timeout(60000);
    this.slow(MAX_ELAPSED_MS);

    before(skipWithoutASecret);

    // Derived from the cases rather than hard-coded, so adding a case of some new profile type cannot
    // silently leave it unrun.  Native array methods rather than lodash: `_` is a Sails global, and it
    // does not exist yet when mocha loads this file to register the tests.
    let profileTypesInOrder = CASES_TO_RUN.reduce((typesSoFar, someCase)=>{
      if(!typesSoFar.includes(someCase.profileType)) { typesSoFar.push(someCase.profileType); }
      return typesSoFar;
    }, []);

    for (let profileType of profileTypesInOrder) {

      describe(profileType, function() {

        for (let testCase of CASES_TO_RUN.filter((someCase)=>{ return someCase.profileType === profileType; })) {

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
              repeats = Array.from({length: REPEATS}, ()=>{
                return generateOnce({ profileType: testCase.profileType, forFleet: !!testCase.forFleet, naturalLanguageInstructions: testCase.instructions });
              });
            });

            for (let repeatIdx = 0; repeatIdx < REPEATS; repeatIdx++) {
              let test = it(`#${repeatIdx + 1}`, async function() {
                assertGeneration({ testCase, expectations: testCase.expect || {}, outcome: await repeats[repeatIdx], repeatIdx, forFleet: !!testCase.forFleet });
              });
              // Read by summary-reporter.js for the per-profile-type tally.
              test.profileType = profileType;
            }
          });
        }
      });
    }
  });
}


if(!ROUTING_ONLY && SUITE !== 'page') {
  describe('configuration profile generator, asked by Fleet', function() {

    this.timeout(60000);
    this.slow(MAX_ELAPSED_MS);

    before(skipWithoutASecret);

    for (let testCase of CASES_TO_RUN.filter((someCase)=>{ return fleetRequestFor(someCase); })) {
      let fleetRequest = fleetRequestFor(testCase);

      describe(`${testCase.id}: ${testCase.instructions}`, function() {

        let repeats = [];
        before(function() {
          repeats = Array.from({length: REPEATS}, ()=>{
            return generateOnce({ platform: 'apple', forFleet: true, naturalLanguageInstructions: testCase.instructions });
          });
        });

        for (let repeatIdx = 0; repeatIdx < REPEATS; repeatIdx++) {
          let test = it(`#${repeatIdx + 1}`, async function() {
            let outcome = await repeats[repeatIdx];
            assert(!outcome.unexpectedError, `unexpected error: ${outcome.unexpectedError && outcome.unexpectedError.message}`);
            if(fleetRequest.routesTo) {
              let routedTo = routeTaken(outcome.generationResult.route, outcome.generationResult.profileType);
              // Recorded before the assertion, so a wrong route is tallied as one even though it fails the test.
              this.test.routing = { expected: fleetRequest.routesTo, actual: routedTo };
              assert.strictEqual(routedTo, fleetRequest.routesTo, `routed to ${routedTo} rather than ${fleetRequest.routesTo}${describeRoute(outcome.generationResult.route)}`);
            }
            // Fleet's server waits no longer for a decline than for a profile.
            assertGeneration({ testCase, expectations: fleetRequest.expect || {}, outcome, repeatIdx, checkLatencyOfDeclines: true, forFleet: true });
          });
          // One bucket rather than ddm and mobileconfig, so that the page's per-type history stays comparable.
          test.profileType = 'apple (Fleet)';
        }
      });
    }
  });
}


if(ROUTING_ONLY) {
  describe('Apple routing', function() {

    this.timeout(60000);

    before(skipWithoutASecret);

    for (let testCase of CASES_TO_RUN.filter((someCase)=>{ return fleetRequestFor(someCase) && fleetRequestFor(someCase).routesTo; })) {
      let fleetRequest = fleetRequestFor(testCase);

      describe(`${testCase.id}: ${testCase.instructions}`, function() {

        let repeats = [];
        before(function() {
          repeats = Array.from({length: REPEATS}, async ()=>{
            try {
              return { route: await sails.helpers.routeAppleConfigurationProfileRequest.with({ naturalLanguageInstructions: testCase.instructions, forFleet: true }) };
            } catch (err) {
              return { unexpectedError: err };
            }
          });
        });

        for (let repeatIdx = 0; repeatIdx < REPEATS; repeatIdx++) {
          it(`#${repeatIdx + 1}`, async function() {
            let {route, unexpectedError} = await repeats[repeatIdx];
            assert(!unexpectedError, `unexpected error: ${unexpectedError && unexpectedError.message}`);
            let routedTo = routeTaken(route, route.profileType);
            this.test.routing = { expected: fleetRequest.routesTo, actual: routedTo };
            assert.strictEqual(routedTo, fleetRequest.routesTo, `routed to ${routedTo} rather than ${fleetRequest.routesTo}${describeRoute(route)}`);
          });
        }
      });
    }
  });
}


/**
 * A mocha `before` hook that skips its suite when there is no secret to generate with.  Without one, every
 * case would fail the same unexplained way, which reads as a broken prompt rather than an unconfigured
 * machine.
 */
function skipWithoutASecret() {
  if(!sails.config.custom.anthropicSecret) {
    console.log('    (skipping: sails.config.custom.anthropicSecret is not set -- re-run with sails_custom__anthropicSecret=… to actually generate anything.)');
    this.skip();
  }
}


/**
 * Assert that one generation did what its case expects: declined when it should, and otherwise generated
 * a profile that passes the case's checks within the time budget.
 *
 * @param  {Dictionary} options
 * @param  {Dictionary} options.testCase
 * @param  {Dictionary} options.expectations  The case's expect block for the request that was made.
 * @param  {Dictionary} options.outcome  What generateOnce resolved to.
 * @param  {Number} options.repeatIdx
 * @param  {Boolean} options.checkLatencyOfDeclines  Whether a decline also has to arrive within the budget.
 * @param  {Boolean} options.forFleet  Whether Fleet asked, which adds the checks for Fleet's own profile rules.
 */
function assertGeneration({testCase, expectations, outcome, repeatIdx, checkLatencyOfDeclines, forFleet}) {
  let {generationResult, unexpectedError, elapsedMs} = outcome;

  // Distinguished from an abstention on purpose: collapsing the two would report a
  // misconfigured anthropicSecret as a model that refused to answer.
  assert(!unexpectedError, `unexpected error: ${unexpectedError && unexpectedError.message}`);

  let {rawResult, generatedProfile, windowsCspAreasProvided, applePayloadTypesProvided, profileType, timings} = generationResult;

  // In every failure message, because a profile written from the wrong part of the schema and one
  // written badly from the right part look identical: the first is a lookup problem, the second a
  // prompt problem.  DDM has no lookup -- its whole schema always goes in -- so there is nothing to
  // say for it.
  let whatTheLookupProvided = '';
  if(profileType === 'csp') {
    let areas = windowsCspAreasProvided || [];
    whatTheLookupProvided = `\n\nCSP areas provided: ${areas.length > 0 ? areas.join(', ') : '(none -- generated without the node reference)'}`;
  } else if(profileType === 'mobileconfig') {
    let payloadTypes = applePayloadTypesProvided || [];
    whatTheLookupProvided = `\n\nPayload types provided: ${payloadTypes.length > 0 ? payloadTypes.join(', ') : '(none -- generated from the full schema, without key descriptions)'}`;
  }

  // The profile goes in the failure message rather than to stdout: a failure you cannot see is a
  // failure you cannot act on, and mocha only shows what the assertion carried.  The model's own
  // output is shown when there is no profile, since a response the generator rejected still says
  // what went wrong.
  let generationForFailureMessage = generatedProfile ?
    `\n\nprofileFilename: ${generatedProfile.profileFilename}` +
    `\ndeliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}` +
    `\n\n${withAdmxTemplatesCollapsed(generatedProfile.profile)}\n\n` +
    `settingsEnforced:\n${util.inspect(generatedProfile.items, { depth: 4, colors: false })}\n` :
    `\n\nModel response:\n${withAdmxTemplatesCollapsed(util.inspect(rawResult, { depth: 4, colors: false }))}\n`;
  let timeTaken = `${elapsedMs}ms (routing ${timings.routingMs}ms, lookups ${timings.configurationMs}ms, generation ${timings.generationMs}ms)`;

  if(expectations.expectFailure) {
    // Content of the wrong type is a broken generation, not the model declining.
    assert(
      !generationResult.contentDidNotMatchProfileType,
      `expected this request to be refused, but it generated content that is not a ${generationResult.profileType} profile` +
      generationForFailureMessage
    );
    assert(
      generationResult.couldNotGenerateProfile,
      'expected this request to be refused, but a profile was generated' +
      (testCase.readByEye ? `\n\nNote on this case: ${testCase.readByEye}` : '') +
      whatTheLookupProvided +
      generationForFailureMessage
    );
    let declineFailures = checkDeclineReason(generationResult.reasonWhyAProfileCouldNotBeGenerated);
    assert.strictEqual(declineFailures.length, 0, declineFailures.join('\n'));
    if(checkLatencyOfDeclines) {
      assert(elapsedMs <= MAX_ELAPSED_MS, `declined as expected, but took ${timeTaken}, over the ${MAX_ELAPSED_MS}ms budget`);
    }
    return;
  }

  assert(
    !generationResult.couldNotGenerateProfile,
    (generationResult.contentDidNotMatchProfileType ?
      `generated content that is not a ${profileType} profile` :
      `abstained but a profile was expected -- reason given: ${JSON.stringify(generationResult.reasonWhyAProfileCouldNotBeGenerated || '(none)')}`) +
    whatTheLookupProvided +
    (generationResult.contentDidNotMatchProfileType ? generationForFailureMessage : '')
  );

  let checkFailures = checkExpectations(expectations, generatedProfile, generationResult.profileType, forFleet);
  assert.strictEqual(
    checkFailures.length, 0,
    `${testCase.canary ? 'CANARY -- ' : ''}${checkFailures.length} check(s) failed:\n` +
    checkFailures.map((checkFailure)=>{ return `  - ${checkFailure}`; }).join('\n') +
    (testCase.readByEye ? `\n\nNote on this case: ${testCase.readByEye}` : '') +
    whatTheLookupProvided +
    generationForFailureMessage
  );

  // Passing the assertions is not the same as being right -- the checks are substrings, and the
  // `readByEye` properties exist precisely because some things no assertion covers.  So print
  // what was generated.  After the assertion rather than before it, so each result is shown
  // exactly once: a failure carries the same material in the message above.
  if(LOG_ALL_GENERATIONS) {
    console.log(
      `\n      ---- ${testCase.id}${REPEATS > 1 ? ` #${repeatIdx + 1}` : ''}${testCase.canary ? ' (CANARY)' : ''} -- ${profileType}, ${elapsedMs}ms ----\n` +
      (testCase.readByEye ? `      CONFIRM BY EYE: ${testCase.readByEye}\n` : '') +
      (whatTheLookupProvided ? `      ${whatTheLookupProvided.trim()}\n` : '') +
      `      profileFilename: ${generatedProfile.profileFilename}\n` +
      `      deliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}\n\n` +
      `${withAdmxTemplatesCollapsed(generatedProfile.profile)}\n\n` +
      `      settingsEnforced:\n${util.inspect(generatedProfile.items, { depth: 4, colors: false })}\n`
    );
  }
  assert(elapsedMs <= MAX_ELAPSED_MS, `generated a passing profile, but took ${timeTaken}, over the ${MAX_ELAPSED_MS}ms budget`);
}


/**
 * Where a request from Fleet was routed: "ddm", "mobileconfig", or "decline" when the router declined it.
 *
 * @param  {Dictionary} route  The router's result.
 * @param  {String} profileType  The profile type generated.
 * @returns {String}
 */
function routeTaken(route, profileType) {
  return route && route.couldNotGenerateProfile ? 'decline' : profileType;
}


/**
 * Why the router chose what it did, for a failure message.
 *
 * @param  {Dictionary} route
 * @returns {String}
 */
function describeRoute(route) {
  if(!route) {
    return '';
  }
  let settingLines = (route.settings || []).map((setting)=>{
    let path = (mapped)=>{ return mapped ? `${mapped.typeName}/${mapped.keyPath}` : 'none'; };
    return `  - ${JSON.stringify(setting.setting)}: declaration ${path(setting.declaration)}, .mobileconfig ${path(setting.payload)}`;
  });
  return `\n\nRouted because: ${route.decidedBecause || '(no reason recorded)'} (router took ${route.routerMs}ms)` +
    (route.reasonWhyAProfileCouldNotBeGenerated ? `\nDecline reason: ${route.reasonWhyAProfileCouldNotBeGenerated}` : '') +
    `\nPlatforms named: ${(route.platformsNamed || []).join(', ') || '(none)'}` +
    `\nSettings the router found:\n${settingLines.join('\n') || '  (none)'}\n`;
}


/**
 * A generated Windows profile with each embedded ADMX template cut down to its size, for printing: the
 * templates are hundreds of KB, and the model did not write them.
 *
 * @param  {String} profile
 * @returns {String}
 */
function withAdmxTemplatesCollapsed(profile) {
  return String(profile || '').replace(/<!\[CDATA\[(\s*(?:<\?xml[^>]*\?>\s*)?<policyDefinitions[\s\S]*?)\]\]>/g, (unusedMatch, admxText)=>{
    return `<![CDATA[…ADMX template, ${Math.round(Buffer.byteLength(admxText, 'utf8') / 1024)} KB…]]>`;
  });
}


/**
 * Generate one profile, and time it.
 *
 * Resolves to what happened rather than rejecting, because these are launched together and awaited one at
 * a time: a repeat that failed before its own test reached it would otherwise surface as an unhandled
 * rejection and take the run with it.
 *
 * A generation the API rate-limited or was too overloaded to serve is retried once, after a pause, and timed
 * from the retry: a 429 or a 529 says how many requests were in flight, not anything about the prompt.
 *
 * @param  {Dictionary} request  What to ask for: a profileType or a platform, and the instructions.
 * @returns {Dictionary}  {generationResult, elapsedMs} or {unexpectedError, elapsedMs}
 */
async function generateOnce(request) {
  for (let attempt = 1; ; attempt++) {
    let startedAt = Date.now();
    try {
      let generationResult = await sails.helpers.generateConfigurationProfile.with(Object.assign({
        baseModel: BASE_MODEL,
        effort: EFFORT,
        disableThinking: DISABLE_THINKING,
        writeProfilesFromSettings: WRITE_PROFILES_FROM_SETTINGS,
      }, request));
      return { generationResult, elapsedMs: Date.now() - startedAt };
    } catch (err) {
      if(attempt === 1 && isWorthRetryingAfterAPause(err)) {
        await new Promise((resolve)=>{ setTimeout(resolve, 3000); });
        continue;
      }
      return { unexpectedError: err, elapsedMs: Date.now() - startedAt };
    }
  }
}
