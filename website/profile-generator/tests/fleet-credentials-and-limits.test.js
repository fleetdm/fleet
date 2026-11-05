/**
 * Who can generate profiles through the Fleet server routes, and how a Fleet Free registration's monthly
 * limit is counted, checked without calling a model: generation is stubbed, and each request goes through
 * the registered route as a Sails virtual request.
 */
const assert = require('assert');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');


describe('profile generator credentials and limits for Fleet servers', function() {

  const PROFILE_URL = '/api/profile-generator/v1/profile';
  const NAME_AND_DESCRIPTION_URL = '/api/profile-generator/v1/name-and-description';

  let originalGenerationHelper;
  let originalPromptHelper;
  let originalLicensePublicKey;
  // What the stubbed generation or model call does: a result, or a function that throws or returns one.
  let stubbedOutcome;
  let stubCalls;
  let createdRegistrationIds = [];

  // Mimics a Sails helper call closely enough for .intercept() and .tolerate() the way the actions use them.
  let stubbedHelper = ()=>{
    return {
      with(inputs) {
        stubCalls.push(inputs);
        let outcome = Promise.resolve().then(()=>{ return _.isFunction(stubbedOutcome) ? stubbedOutcome() : stubbedOutcome; });
        outcome.intercept = (handler)=>{ return outcome.catch((err)=>{ throw handler(err); }); };
        outcome.tolerate = (exitName, handler)=>{
          return outcome.catch((err)=>{
            if(err.exit !== exitName) {
              throw err;
            }
            return handler(err);
          });
        };
        return outcome;
      },
    };
  };

  let licenseSigningKeys = crypto.generateKeyPairSync('ec', { namedCurve: 'P-256' });
  let licenseKey = (claims, signingKey = licenseSigningKeys.privateKey)=>{
    return require('jsonwebtoken').sign(
      Object.assign({ iss: 'Fleet Device Management Inc.', sub: 'credentials-and-limits-tests', tier: 'premium', devices: 10, exp: Math.floor(Date.now() / 1000) + 3600 }, claims),
      signingKey,
      { algorithm: 'ES256' }
    );
  };
  let expiredLicenseKey = licenseKey({ exp: Math.floor(Date.now() / 1000) - 60 });

  let generatedResult = { profileType: 'csp', couldNotGenerateProfile: false, generatedProfile: { profile: '<Replace/>', profileFilename: 'x.xml', deliveryNotes: '', items: [], additionalProfiles: [] } };
  let namedResult = { name: 'Block removable storage', description: 'Stops writes to USB drives.' };
  let thisMonth = ()=>{ return new Date().toISOString().slice(0, 7); };
  let firstOfNextMonth = ()=>{
    let now = new Date();
    return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1)).toISOString();
  };

  before(function() {
    originalGenerationHelper = sails.helpers.generateConfigurationProfile;
    originalPromptHelper = sails.helpers.ai.prompt;
    sails.helpers.generateConfigurationProfile = stubbedHelper();
    sails.helpers.ai.prompt = stubbedHelper();
    originalLicensePublicKey = sails.config.custom.licenseKeyGeneratorPublicKey;
    sails.config.custom.licenseKeyGeneratorPublicKey = licenseSigningKeys.publicKey.export({ type: 'spki', format: 'pem' });
  });

  after(async function() {
    sails.helpers.generateConfigurationProfile = originalGenerationHelper;
    sails.helpers.ai.prompt = originalPromptHelper;
    sails.config.custom.licenseKeyGeneratorPublicKey = originalLicensePublicKey;
    if(createdRegistrationIds.length > 0) {
      await ProfileGeneratorRegistration.destroy({ id: { in: createdRegistrationIds } });
    }
  });

  beforeEach(function() {
    stubCalls = [];
    stubbedOutcome = undefined;
  });

  // A request with no marketingAttribution cookie makes the custom hook set one, which a virtual response
  // cannot do, so every request carries one.
  let post = (url, data, fleetServerSecret)=>{
    let headers = { cookie: 'marketingAttribution=%7B%7D' };
    if(fleetServerSecret) {
      headers.authorization = `Bearer ${fleetServerSecret}`;
    }
    return new Promise((resolve)=>{
      sails.request({ method: 'POST', url, data, headers }, (err, res, body)=>{
        resolve(err ? { status: err.status, body: err.body } : { status: res.statusCode, body });
      });
    });
  };
  let requestProfile = (fleetServerSecret, data)=>{
    return post(PROFILE_URL, _.extend({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' }, data), fleetServerSecret);
  };
  let requestNameAndDescription = (fleetServerSecret, data)=>{
    return post(NAME_AND_DESCRIPTION_URL, _.extend({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' }, data), fleetServerSecret);
  };

  // A registration made directly, since registering is covered by its own tests.
  let createRegistration = async (usage)=>{
    let fleetServerSecret = crypto.randomBytes(32).toString('hex');
    let registration = await ProfileGeneratorRegistration.create(_.extend({
      fleetServerUrl: `https://fleet-${crypto.randomBytes(6).toString('hex')}.example.com`,
      fleetServerSecretHash: crypto.createHash('sha256').update(fleetServerSecret).digest('hex'),
      registeredFromIpAddress: '203.0.113.7',
    }, usage)).fetch();
    createdRegistrationIds.push(registration.id);
    return { id: registration.id, fleetServerSecret };
  };
  let usageOf = async (registration)=>{
    let { usageMonth, profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth } = await ProfileGeneratorRegistration.findOne({ id: registration.id });
    return { usageMonth, profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth };
  };
  let usedThisMonth = (profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth)=>{
    return { usageMonth: thisMonth(), profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth };
  };
  // What the prompt helper throws when the AI service gives no answer.
  let aiServiceDown = ()=>{ throw _.extend(new Error('overloaded'), { statusCode: 529, llmDidNotAnswer: true }); };

  describe('credentials', function() {

    it('generates without a limit, counting nothing, for a premium, trial, or basic license', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 15, nameAndDescriptionRequestsThisMonth: 45 });
      for (let tier of ['premium', 'trial', 'basic']) {
        stubbedOutcome = generatedResult;
        // The registration's secret is sent too, and is at its limit: a valid license takes precedence.
        let response = await requestProfile(registration.fleetServerSecret, { fleetLicenseKey: licenseKey({ tier }) });
        assert.strictEqual(response.status, 200, `expected a 200 for a ${tier} license`);
        stubbedOutcome = namedResult;
        response = await requestNameAndDescription(registration.fleetServerSecret, { fleetLicenseKey: licenseKey({ tier }) });
        assert.strictEqual(response.status, 200, `expected a 200 for a ${tier} license`);
      }
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(15, 45));
    });

    it('accepts the development license Fleet serves with --dev_license', async function() {
      let serveGo = fs.readFileSync(path.resolve(__dirname, '../../../cmd/fleet/serve.go'), 'utf8');
      let developmentLicenseKey = serveGo.match(/if devLicense \{[^}]*config\.License\.Key = "([^"]+)"/)[1];
      let originalNow = Date.now;
      // Pinned inside the license's validity, so this keeps passing after it expires and is replaced.
      let { iat } = require('jsonwebtoken').decode(developmentLicenseKey);
      sails.config.custom.licenseKeyGeneratorPublicKey = fs.readFileSync(path.resolve(__dirname, '../../../ee/server/licensing/pubkey.pem'), 'utf8');
      Date.now = ()=>{ return (iat + 60) * 1000; };
      let response;
      try {
        stubbedOutcome = generatedResult;
        response = await requestProfile(undefined, { fleetLicenseKey: developmentLicenseKey });
      } finally {
        Date.now = originalNow;
        sails.config.custom.licenseKeyGeneratorPublicKey = licenseSigningKeys.publicKey.export({ type: 'spki', format: 'pem' });
      }
      assert.strictEqual(response.status, 200);
    });

    it('is a 401 for an expired license alone, and limits it like Fleet Free with a registration secret', async function() {
      let response = await requestProfile(undefined, { fleetLicenseKey: expiredLicenseKey });
      assert.strictEqual(response.status, 401);

      let registration = await createRegistration();
      stubbedOutcome = generatedResult;
      response = await requestProfile(registration.fleetServerSecret, { fleetLicenseKey: expiredLicenseKey });
      assert.strictEqual(response.status, 200);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(1, 0));
      assert.strictEqual(stubCalls.length, 1);
    });

    it('is a 401, counting nothing, without a valid license or a known registration secret', async function() {
      let registration = await createRegistration();
      let unusableLicenseKeys = [
        licenseKey({ tier: 'free' }),
        licenseKey({ iss: 'Someone else' }),
        licenseKey({}, crypto.generateKeyPairSync('ec', { namedCurve: 'P-256' }).privateKey),
        'not a license key',
      ];
      for (let fleetLicenseKey of unusableLicenseKeys) {
        let response = await requestProfile(undefined, { fleetLicenseKey });
        assert.strictEqual(response.status, 401);
      }
      for (let fleetServerSecret of [undefined, crypto.randomBytes(32).toString('hex')]) {
        assert.strictEqual((await requestProfile(fleetServerSecret)).status, 401);
        assert.strictEqual((await requestNameAndDescription(fleetServerSecret)).status, 401);
      }
      // A secret that is right but not sent as a bearer token.
      let response = await post(PROFILE_URL, { platform: 'windows', naturalLanguageInstructions: 'Block removable storage', fleetServerSecret: registration.fleetServerSecret });
      assert.strictEqual(response.status, 401);
      assert.strictEqual(stubCalls.length, 0);
      assert.deepStrictEqual(await usageOf(registration), { usageMonth: '', profileGenerationsThisMonth: 0, nameAndDescriptionRequestsThisMonth: 0 });
    });

  });

  describe('Fleet Free profile limit', function() {

    it('allows the 15th profile in a month and is a 429 for the 16th, until the 1st of next month (UTC)', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 14 });
      stubbedOutcome = generatedResult;
      let fifteenth = await requestProfile(registration.fleetServerSecret);
      assert.strictEqual(fifteenth.status, 200);
      let sixteenth = await requestProfile(registration.fleetServerSecret);
      assert.strictEqual(sixteenth.status, 429);
      assert.deepStrictEqual(sixteenth.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth() });
      assert.strictEqual(stubCalls.length, 1);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(15, 0));
    });

    it('counts a decline and an answer it could not use, but not a request rejected for bad input', async function() {
      let registration = await createRegistration();
      stubbedOutcome = { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: 'This request needs two profiles.' };
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 400);
      stubbedOutcome = ()=>{ throw new Error('Expecting JSON result from LLM, but it could not be parsed.'); };
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 500);
      stubbedOutcome = { profileType: 'mobileconfig', couldNotGenerateProfile: true, contentDidNotMatchProfileType: true };
      assert.strictEqual((await requestProfile(registration.fleetServerSecret, { platform: 'apple' })).status, 500);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(3, 0));

      let tooLong = await requestProfile(registration.fleetServerSecret, { naturalLanguageInstructions: 'a'.repeat(2001) });
      assert.strictEqual(tooLong.status, 400);
      let unknownPlatform = await requestProfile(registration.fleetServerSecret, { platform: 'linux' });
      assert.strictEqual(unknownPlatform.status, 400);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(3, 0));
    });

    it('gives a request back when the AI service never answers', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 14 });
      stubbedOutcome = aiServiceDown;
      let response = await requestProfile(registration.fleetServerSecret);
      assert.strictEqual(response.status, 500);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(14, 0));
      // So the 15th is still available.
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 200);
    });

    it('lets exactly one of several concurrent requests through at 14 used', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 14 });
      stubbedOutcome = generatedResult;
      let responses = await Promise.all(_.times(5, ()=>{ return requestProfile(registration.fleetServerSecret); }));
      assert.deepStrictEqual(_.pluck(responses, 'status').sort(), [200, 429, 429, 429, 429]);
      assert.strictEqual(stubCalls.length, 1);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(15, 0));
    });

    it('starts the counters from zero with the first request in a new month', async function() {
      let registration = await createRegistration({ usageMonth: '2026-01', profileGenerationsThisMonth: 15, nameAndDescriptionRequestsThisMonth: 45 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 200);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(1, 0));
    });

    it('tells a failure with no answer from one with an unusable answer through the real generation and prompt helpers', async function() {
      // Only the call to the AI service is faked, so this checks the flag the prompt helper sets survives
      // the generation helper on its way to the route.
      let originalHttpPost = sails.helpers.http.post;
      let originalAnthropicSecret = sails.config.custom.anthropicSecret;
      let fakeAnthropic;
      sails.helpers.generateConfigurationProfile = originalGenerationHelper;
      sails.helpers.ai.prompt = originalPromptHelper;
      sails.config.custom.anthropicSecret = 'not-a-real-secret';
      sails.helpers.http.post = function() {
        let outcome = Promise.resolve().then(fakeAnthropic);
        let interceptors = [];
        let deferred = outcome.catch((err)=>{
          let interceptor = _.find(interceptors, (candidate)=>{ return !candidate.exitName || candidate.exitName === err.exit; });
          throw interceptor ? interceptor.handler(err) : err;
        });
        deferred.intercept = (exitName, handler)=>{
          interceptors.push(handler ? { exitName, handler } : { handler: exitName });
          return deferred;
        };
        return deferred;
      };
      try {
        let registration = await createRegistration();
        // A 400 rather than a 429 or 5xx, so the generation helper does not wait a second to retry it.
        fakeAnthropic = ()=>{ throw { exit: 'non200Response', raw: { statusCode: 400, body: '{"type":"error"}' } }; };
        assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 500);
        assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 0));
        fakeAnthropic = ()=>{ throw new Error('ECONNRESET'); };
        assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 500);
        assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 0));
        fakeAnthropic = ()=>{ return { stop_reason: 'end_turn', content: [{ type: 'text', text: 'Not JSON at all.' }] }; };// eslint-disable-line camelcase
        assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 500);
        assert.deepStrictEqual(await usageOf(registration), usedThisMonth(1, 0));
      } finally {
        sails.helpers.http.post = originalHttpPost;
        sails.config.custom.anthropicSecret = originalAnthropicSecret;
        sails.helpers.generateConfigurationProfile = stubbedHelper();
        sails.helpers.ai.prompt = stubbedHelper();
      }
    });

    it('never moves a row back to an earlier month, as a request read just before midnight UTC would', async function() {
      // A newer request already moved the row into the next month while this one was starting.
      let now = new Date();
      let nextMonth = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1)).toISOString().slice(0, 7);
      let monthAfterNext = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 2, 1)).toISOString();
      let registration = await createRegistration({ usageMonth: nextMonth, profileGenerationsThisMonth: 14 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 200);
      assert.deepStrictEqual(await usageOf(registration), { usageMonth: nextMonth, profileGenerationsThisMonth: 15, nameAndDescriptionRequestsThisMonth: 0 });
      let limited = await requestProfile(registration.fleetServerSecret);
      assert.strictEqual(limited.status, 429);
      assert.strictEqual(limited.body.resetsAt, monthAfterNext);
    });

    it('gives nothing back to a month that has already ended', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 3 });
      await sails.helpers.profileGenerator.giveBackOneRequest.with({ registrationId: registration.id, requestType: 'profile', usageMonth: '2026-01' });
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(3, 0));
    });

  });

  describe('Fleet Free names and descriptions', function() {

    it('counts names and descriptions separately from profiles', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 14 });
      stubbedOutcome = namedResult;
      for (let i = 0; i < 3; i++) {
        assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 200);
      }
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(14, 3));
    });

    it('is a 429 once the profile limit is reached', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 15 });
      let response = await requestNameAndDescription(registration.fleetServerSecret);
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth() });
      assert.strictEqual(stubCalls.length, 0);
    });

    it('is a 429 after 45 in a month, which leaves profiles available', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), nameAndDescriptionRequestsThisMonth: 44 });
      stubbedOutcome = namedResult;
      assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 200);
      let response = await requestNameAndDescription(registration.fleetServerSecret);
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth() });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 200);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(1, 45));
    });

    it('gives a timed-out request back if the AI service then fails without answering', async function() {
      this.timeout(15000);
      let registration = await createRegistration();
      // Fails after the route has stopped waiting at 8 seconds.
      stubbedOutcome = ()=>{
        return new Promise((resolve, reject)=>{ setTimeout(()=>{ reject(_.extend(new Error('overloaded'), { statusCode: 529, llmDidNotAnswer: true })); }, 8500); });
      };
      assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 500);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 1), 'counted while the AI service might still answer');
      await new Promise((resolve)=>{ setTimeout(resolve, 1000); });
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 0));
    });

    it('gives a request back when the AI service never answers, and counts a refusal', async function() {
      let registration = await createRegistration();
      stubbedOutcome = aiServiceDown;
      assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 500);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 0));
      stubbedOutcome = ()=>{ throw { exit: 'refused' }; };
      assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 400);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(0, 1));
    });

  });

});
