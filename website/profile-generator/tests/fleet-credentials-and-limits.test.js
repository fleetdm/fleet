/**
 * Who can generate profiles through the Fleet server routes, and how the monthly limits of a Fleet Free
 * registration and a Fleet Premium license are counted, checked without calling a model: generation is stubbed, and each request goes through
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
  let createdUsageIds = [];

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

  // Every license these tests sign is issued to this organization, so their usage rows can be removed after.
  const LICENSE_ORGANIZATION = 'credentials-and-limits-tests';
  let licenseSigningKeys = crypto.generateKeyPairSync('ec', { namedCurve: 'P-256' });
  // A nonce makes each call a different license, with its own usage, even within one second.
  let licenseKey = (claims, signingKey = licenseSigningKeys.privateKey)=>{
    return require('jsonwebtoken').sign(
      Object.assign({ iss: 'Fleet Device Management Inc.', sub: LICENSE_ORGANIZATION, tier: 'premium', devices: 10, exp: Math.floor(Date.now() / 1000) + 3600, nonce: crypto.randomBytes(8).toString('hex') }, claims),
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
    let usageIds = _.pluck(await ProfileGeneratorUsage.find({ organization: LICENSE_ORGANIZATION }), 'id').concat(createdUsageIds);
    if(usageIds.length > 0) {
      await ProfileGeneratorUsage.destroy({ id: { in: usageIds } });
    }
  });

  beforeEach(function() {
    stubCalls = [];
    stubbedOutcome = undefined;
  });

  // A request with no marketingAttribution cookie makes the custom hook set one, which a virtual response
  // cannot do, so every request carries one.
  let post = (url, data, fleetServerSecret, extraHeaders)=>{
    let headers = { cookie: 'marketingAttribution=%7B%7D' };
    if(fleetServerSecret) {
      headers.authorization = `Bearer ${fleetServerSecret}`;
    }
    _.extend(headers, extraHeaders);
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
    let registration = await ProfileGeneratorUsage.create(_.extend({
      credentialType: 'registration',
      fleetServerUrl: `https://fleet-${crypto.randomBytes(6).toString('hex')}.example.com`,
      credentialHash: crypto.createHash('sha256').update(fleetServerSecret).digest('hex'),
      registeredFromIpAddress: '203.0.113.7',
    }, usage)).fetch();
    createdUsageIds.push(registration.id);
    return { id: registration.id, fleetServerSecret };
  };
  let usageOf = async (registration)=>{
    let { usageMonth, profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth } = await ProfileGeneratorUsage.findOne({ id: registration.id });
    return { usageMonth, profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth };
  };
  let usedThisMonth = (profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth)=>{
    return { usageMonth: thisMonth(), profileGenerationsThisMonth, nameAndDescriptionRequestsThisMonth };
  };
  let licenseKeyHashOf = (fleetLicenseKey)=>{
    return crypto.createHash('sha256').update(fleetLicenseKey.split('.').slice(0, 2).join('.')).digest('hex');
  };
  // A license's usage made directly, as its first request would make it.
  let createLicenseUsage = async (fleetLicenseKey, usage)=>{
    let licenseUsage = await ProfileGeneratorUsage.create(_.extend({ credentialType: 'license', credentialHash: licenseKeyHashOf(fleetLicenseKey), organization: LICENSE_ORGANIZATION }, usage)).fetch();
    createdUsageIds.push(licenseUsage.id);
    return licenseUsage;
  };
  let licenseUsageOf = async (fleetLicenseKey)=>{
    let licenseUsage = await ProfileGeneratorUsage.findOne({ credentialHash: licenseKeyHashOf(fleetLicenseKey) });
    return licenseUsage && _.pick(licenseUsage, 'usageMonth', 'profileGenerationsThisMonth', 'nameAndDescriptionRequestsThisMonth');
  };
  // What the prompt helper throws when the AI service gives no answer.
  let aiServiceDown = ()=>{ throw _.extend(new Error('overloaded'), { statusCode: 529, llmDidNotAnswer: true }); };

  describe('credentials', function() {

    it('counts a premium, trial, or basic license on its own usage, not on the registration it also sends', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), profileGenerationsThisMonth: 15, nameAndDescriptionRequestsThisMonth: 45 });
      for (let tier of ['premium', 'trial', 'basic']) {
        let fleetLicenseKey = licenseKey({ tier });
        stubbedOutcome = generatedResult;
        // The registration's secret is sent too, and is at its limit: a valid license takes precedence.
        let response = await requestProfile(registration.fleetServerSecret, { fleetLicenseKey });
        assert.strictEqual(response.status, 200, `expected a 200 for a ${tier} license`);
        stubbedOutcome = namedResult;
        response = await requestNameAndDescription(registration.fleetServerSecret, { fleetLicenseKey });
        assert.strictEqual(response.status, 200, `expected a 200 for a ${tier} license`);
        assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(1, 1), `expected the ${tier} license's own usage`);
      }
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(15, 45));
    });

    it('records the organization a license was issued to', async function() {
      let fleetLicenseKey = licenseKey({});
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      let licenseUsage = await ProfileGeneratorUsage.findOne({ credentialHash: licenseKeyHashOf(fleetLicenseKey) });
      assert.strictEqual(licenseUsage.organization, LICENSE_ORGANIZATION);
    });

    it('accepts the development license Fleet serves with --dev_license, and limits it like any license', async function() {
      let serveGo = fs.readFileSync(path.resolve(__dirname, '../../../cmd/fleet/serve.go'), 'utf8');
      let developmentLicenseKey = serveGo.match(/if devLicense \{[^}]*config\.License\.Key = "([^"]+)"/)[1];
      let originalNow = Date.now;
      // Pinned inside the license's validity, so this keeps passing after it expires and is replaced.
      let { iat } = require('jsonwebtoken').decode(developmentLicenseKey);
      // A local website run with --dev_license may already have counted it, so it starts from zero here and its
      // usage is put back after.
      let usageBefore = await ProfileGeneratorUsage.findOne({ credentialHash: licenseKeyHashOf(developmentLicenseKey) });
      sails.config.custom.licenseKeyGeneratorPublicKey = fs.readFileSync(path.resolve(__dirname, '../../../ee/server/licensing/pubkey.pem'), 'utf8');
      Date.now = ()=>{ return (iat + 60) * 1000; };
      let response;
      let limited;
      let usageMonth;
      try {
        if(usageBefore) {
          await ProfileGeneratorUsage.updateOne({ id: usageBefore.id }).set({ usageMonth: '', profileGenerationsThisMonth: 0, nameAndDescriptionRequestsThisMonth: 0 });
        }
        usageMonth = new Date().toISOString().slice(0, 7);
        stubbedOutcome = generatedResult;
        response = await requestProfile(undefined, { fleetLicenseKey: developmentLicenseKey });
        await ProfileGeneratorUsage.updateOne({ credentialHash: licenseKeyHashOf(developmentLicenseKey) }).set({ usageMonth, profileGenerationsThisMonth: 300 });
        limited = await requestProfile(undefined, { fleetLicenseKey: developmentLicenseKey });
      } finally {
        Date.now = originalNow;
        sails.config.custom.licenseKeyGeneratorPublicKey = licenseSigningKeys.publicKey.export({ type: 'spki', format: 'pem' });
        if(usageBefore) {
          await ProfileGeneratorUsage.updateOne({ id: usageBefore.id }).set(_.pick(usageBefore, 'usageMonth', 'profileGenerationsThisMonth', 'nameAndDescriptionRequestsThisMonth'));
        } else {
          await ProfileGeneratorUsage.destroy({ credentialHash: licenseKeyHashOf(developmentLicenseKey) });
        }
      }
      assert.strictEqual(response.status, 200);
      assert.strictEqual(limited.status, 429);
      assert.strictEqual(limited.body.limit, 300);
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
      // The scheme is case-insensitive.
      stubbedOutcome = generatedResult;
      let lowercase = await post(PROFILE_URL, { platform: 'windows', naturalLanguageInstructions: 'Block removable storage' }, undefined, { authorization: `bearer ${registration.fleetServerSecret}` });
      assert.strictEqual(lowercase.status, 200);
      stubCalls = [];
      await ProfileGeneratorUsage.updateOne({ id: registration.id }).set({ usageMonth: '', profileGenerationsThisMonth: 0 });

      // A secret that is right but not sent as a bearer token.
      let response = await post(PROFILE_URL, { platform: 'windows', naturalLanguageInstructions: 'Block removable storage', fleetServerSecret: registration.fleetServerSecret });
      assert.strictEqual(response.status, 401);
      assert.strictEqual(stubCalls.length, 0);
      assert.deepStrictEqual(await usageOf(registration), { usageMonth: '', profileGenerationsThisMonth: 0, nameAndDescriptionRequestsThisMonth: 0 });
    });

    it('is a 401 for a license key sent as a registration secret, with or without its signature', async function() {
      // A license's stored hash is of its header and payload, which is not a secret on its own.
      let fleetLicenseKey = licenseKey({});
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      stubCalls = [];
      for (let fleetServerSecret of [fleetLicenseKey.split('.').slice(0, 2).join('.'), fleetLicenseKey]) {
        assert.strictEqual((await requestProfile(fleetServerSecret)).status, 401);
        assert.strictEqual((await requestNameAndDescription(fleetServerSecret)).status, 401);
      }
      assert.strictEqual(stubCalls.length, 0);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(1, 0));
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
      assert.deepStrictEqual(sixteenth.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 15 });
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
      await sails.helpers.profileGenerator.giveBackOneRequest.with({ usageId: registration.id, requestType: 'profile', usageMonth: '2026-01' });
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
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 15 });
      assert.strictEqual(stubCalls.length, 0);
    });

    it('is a 429 after 45 in a month, which leaves profiles available', async function() {
      let registration = await createRegistration({ usageMonth: thisMonth(), nameAndDescriptionRequestsThisMonth: 44 });
      stubbedOutcome = namedResult;
      assert.strictEqual((await requestNameAndDescription(registration.fleetServerSecret)).status, 200);
      let response = await requestNameAndDescription(registration.fleetServerSecret);
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 15 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(registration.fleetServerSecret)).status, 200);
      assert.deepStrictEqual(await usageOf(registration), usedThisMonth(1, 45));
    });

    it('times out 8 seconds after the request arrived, including the time spent counting it', async function() {
      this.timeout(15000);
      let registration = await createRegistration();
      let originalCountOneRequest = sails.helpers.profileGenerator.countOneRequest;
      sails.helpers.profileGenerator.countOneRequest = {
        with(inputs) {
          let counted = new Promise((resolve)=>{ setTimeout(resolve, 2000); }).then(()=>{ return originalCountOneRequest.with(inputs); });
          counted.intercept = (exitName, handler)=>{ return counted.catch((err)=>{ throw err.exit === exitName ? handler(err) : err; }); };
          return counted;
        },
      };
      stubbedOutcome = ()=>{ return new Promise(()=>{}); };
      let startedAt = Date.now();
      let response;
      try {
        response = await requestNameAndDescription(registration.fleetServerSecret);
      } finally {
        sails.helpers.profileGenerator.countOneRequest = originalCountOneRequest;
      }
      assert.strictEqual(response.status, 500);
      assert.ok(Date.now() - startedAt < 9000, `answered after ${Date.now() - startedAt}ms`);
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

  describe('Fleet Premium limit', function() {

    it('allows the 300th profile in a month and is a 429 for the 301st, until the 1st of next month (UTC)', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), profileGenerationsThisMonth: 299 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      let response = await requestProfile(undefined, { fleetLicenseKey });
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 300 });
      assert.strictEqual(stubCalls.length, 1);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(300, 0));
    });

    it('limits a trial and a basic license the same way', async function() {
      for (let tier of ['trial', 'basic']) {
        let fleetLicenseKey = licenseKey({ tier });
        await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), profileGenerationsThisMonth: 300 });
        let response = await requestProfile(undefined, { fleetLicenseKey });
        assert.strictEqual(response.status, 429, `expected a 429 for a ${tier} license`);
        assert.strictEqual(response.body.limit, 300);
      }
      assert.strictEqual(stubCalls.length, 0);
    });

    it('shares one count between Fleet servers that send the same license', async function() {
      let fleetLicenseKey = licenseKey({});
      let firstFleetServer = await createRegistration();
      let secondFleetServer = await createRegistration();
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(firstFleetServer.fleetServerSecret, { fleetLicenseKey })).status, 200);
      assert.strictEqual((await requestProfile(secondFleetServer.fleetServerSecret, { fleetLicenseKey })).status, 200);
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(3, 0));
      assert.strictEqual(await ProfileGeneratorUsage.count({ credentialHash: licenseKeyHashOf(fleetLicenseKey) }), 1);
    });

    it('shares one count between two signatures of the same license', async function() {
      // ES256 signatures are randomized, so signing the same claims again gives a different key for the same license.
      let iat = Math.floor(Date.now() / 1000);
      let claims = { iat, exp: iat + 3600, nonce: crypto.randomBytes(8).toString('hex') };
      let fleetLicenseKey = licenseKey(claims);
      let resignedLicenseKey = licenseKey(claims);
      assert.notStrictEqual(resignedLicenseKey, fleetLicenseKey);
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey: resignedLicenseKey })).status, 200);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(2, 0));
    });

    it('counts a decline, but not a request rejected for bad input or one the AI service never answers', async function() {
      let fleetLicenseKey = licenseKey({});
      stubbedOutcome = { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: 'This request needs two profiles.' };
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 400);
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey, naturalLanguageInstructions: 'a'.repeat(2001) })).status, 400);
      stubbedOutcome = aiServiceDown;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 500);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(1, 0));
    });

    it('lets exactly one of several concurrent requests through at 299 used', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), profileGenerationsThisMonth: 299 });
      stubbedOutcome = generatedResult;
      let responses = await Promise.all(_.times(5, ()=>{ return requestProfile(undefined, { fleetLicenseKey }); }));
      assert.deepStrictEqual(_.pluck(responses, 'status').sort(), [200, 429, 429, 429, 429]);
      assert.strictEqual(stubCalls.length, 1);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(300, 0));
    });

    it('answers two concurrent first requests for a license and keeps one usage row', async function() {
      let fleetLicenseKey = licenseKey({});
      // Fleet asks for a profile and for its name and description at the same time, so the two stubs answer
      // differently here.
      stubbedOutcome = generatedResult;
      let originalPrompt = sails.helpers.ai.prompt;
      sails.helpers.ai.prompt = { with() { let named = Promise.resolve(namedResult); named.tolerate = ()=>{ return named; }; return named; } };
      let responses;
      try {
        responses = await Promise.all([
          requestProfile(undefined, { fleetLicenseKey }),
          requestNameAndDescription(undefined, { fleetLicenseKey }),
          requestProfile(undefined, { fleetLicenseKey }),
        ]);
      } finally {
        sails.helpers.ai.prompt = originalPrompt;
      }
      assert.deepStrictEqual(_.pluck(responses, 'status'), [200, 200, 200]);
      assert.strictEqual(await ProfileGeneratorUsage.count({ credentialHash: licenseKeyHashOf(fleetLicenseKey) }), 1);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(2, 1));
    });

    it('starts the counters from zero with the first request in a new month', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: '2026-01', profileGenerationsThisMonth: 300, nameAndDescriptionRequestsThisMonth: 900 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(1, 0));
    });

  });

  describe('Fleet Premium names and descriptions', function() {

    it('counts names and descriptions separately from profiles, and gives back one the AI service never answers', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), profileGenerationsThisMonth: 299 });
      stubbedOutcome = namedResult;
      for (let i = 0; i < 3; i++) {
        assert.strictEqual((await requestNameAndDescription(undefined, { fleetLicenseKey })).status, 200);
      }
      stubbedOutcome = aiServiceDown;
      assert.strictEqual((await requestNameAndDescription(undefined, { fleetLicenseKey })).status, 500);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(299, 3));
    });

    it('is a 429 with the profile limit once the profile limit is reached', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), profileGenerationsThisMonth: 300 });
      let response = await requestNameAndDescription(undefined, { fleetLicenseKey });
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 300 });
      assert.strictEqual(stubCalls.length, 0);
    });

    it('is a 429 with the profile limit after 900 in a month, which leaves profiles available', async function() {
      let fleetLicenseKey = licenseKey({});
      await createLicenseUsage(fleetLicenseKey, { usageMonth: thisMonth(), nameAndDescriptionRequestsThisMonth: 899 });
      stubbedOutcome = namedResult;
      assert.strictEqual((await requestNameAndDescription(undefined, { fleetLicenseKey })).status, 200);
      let response = await requestNameAndDescription(undefined, { fleetLicenseKey });
      assert.strictEqual(response.status, 429);
      assert.deepStrictEqual(response.body, { error: 'generationLimitReached', resetsAt: firstOfNextMonth(), limit: 300 });
      stubbedOutcome = generatedResult;
      assert.strictEqual((await requestProfile(undefined, { fleetLicenseKey })).status, 200);
      assert.deepStrictEqual(await licenseUsageOf(fleetLicenseKey), usedThisMonth(1, 900));
    });

  });

});
