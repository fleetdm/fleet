/**
 * What Fleet's server can rely on from the profile generator's endpoints over HTTP, checked without calling a
 * model: the generation and the model are stubbed, and each request goes through the registered route as a
 * Sails virtual request, so routing, input validation, status codes and response bodies are all the real ones.
 */
const assert = require('assert');


describe('profile generator endpoints over HTTP', function() {

  let originalGenerationHelper;
  let originalPromptHelper;
  // What the stubbed generation or model call does: a result, a function that throws, or a promise that never settles.
  let stubbedOutcome;
  let stubCalls;

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

  before(function() {
    originalGenerationHelper = sails.helpers.generateConfigurationProfile;
    originalPromptHelper = sails.helpers.ai.prompt;
    sails.helpers.generateConfigurationProfile = stubbedHelper();
    sails.helpers.ai.prompt = stubbedHelper();
  });

  after(function() {
    sails.helpers.generateConfigurationProfile = originalGenerationHelper;
    sails.helpers.ai.prompt = originalPromptHelper;
  });

  beforeEach(function() {
    stubCalls = [];
    stubbedOutcome = undefined;
  });

  // A request with no marketingAttribution cookie makes the custom hook set one, which a virtual response
  // cannot do, so every request carries one.
  let post = (url, data)=>{
    return new Promise((resolve)=>{
      sails.request({ method: 'POST', url, data, headers: { cookie: 'marketingAttribution=%7B%7D' } }, (err, res, body)=>{
        resolve(err ? { status: err.status, body: err.body } : { status: res.statusCode, body });
      });
    });
  };
  let requestProfile = (data)=>{ return post('/api/v1/get-llm-generated-configuration-profile', _.extend({ forFleet: true }, data)); };
  let requestNameAndDescription = (data)=>{ return post('/api/v1/get-llm-generated-configuration-profile-name-and-description', data); };
  let generatedResult = { profileType: 'ddm', couldNotGenerateProfile: false, generatedProfile: { profile: '{}', profileFilename: 'x.json', deliveryNotes: '', items: [], additionalProfiles: [] } };

  it('is a 400 when neither or both of profileType and platform are given', async function() {
    for (let data of [{}, { profileType: 'ddm', platform: 'apple' }]) {
      let response = await requestProfile(_.extend({ naturalLanguageInstructions: 'Turn off AirDrop' }, data));
      assert.strictEqual(response.status, 400);
      assert.strictEqual(response.body.error, 'invalidInput');
    }
    assert.strictEqual(stubCalls.length, 0);
  });

  it('is a 400 for a platform it does not know', async function() {
    let response = await requestProfile({ platform: 'linux', naturalLanguageInstructions: 'Turn off the camera' });
    assert.strictEqual(response.status, 400);
    assert.strictEqual(stubCalls.length, 0);
  });

  it('counts instructions in code points: 2,000 are accepted and 2,001 are a 400', async function() {
    stubbedOutcome = generatedResult;
    let accepted = await requestProfile({ platform: 'apple', naturalLanguageInstructions: '😀'.repeat(2000) });
    assert.strictEqual(accepted.status, 200);
    assert.strictEqual(accepted.body.profileType, 'ddm');
    let rejected = await requestProfile({ platform: 'apple', naturalLanguageInstructions: '😀'.repeat(2001) });
    assert.strictEqual(rejected.status, 400);
    assert.strictEqual(rejected.body.error, 'invalidInput');
    assert.strictEqual(stubCalls.length, 1);
  });

  it('names the profile type first in a generated profile', async function() {
    stubbedOutcome = generatedResult;
    let response = await requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Require a passcode' });
    assert.strictEqual(response.status, 200);
    assert.strictEqual(Object.keys(response.body)[0], 'profileType');
    assert.strictEqual(response.body.profileFilename, 'x.json');
  });

  it('is a 400 with the reason when the model declines', async function() {
    stubbedOutcome = { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: 'This request needs two profiles.' };
    let response = await requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 400);
    assert.deepStrictEqual(response.body, { error: 'couldNotGenerateProfile', reason: 'This request needs two profiles.' });

    stubbedOutcome = { couldNotGenerateProfile: true };
    response = await requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 400);
    assert.ok(response.body.reason, 'a decline with no reason from the model still gets one');
  });

  it('is a 500, not a decline, when generation fails or produces the wrong kind of profile', async function() {
    stubbedOutcome = ()=>{ throw new Error('overloaded'); };
    let response = await requestProfile({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' });
    assert.strictEqual(response.status, 500);

    stubbedOutcome = { profileType: 'mobileconfig', couldNotGenerateProfile: true, contentDidNotMatchProfileType: true };
    response = await requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 500);
  });

  it('names and describes a profile, ignoring the forFleet that Fleet also sends', async function() {
    stubbedOutcome = { name: 'Disable AirDrop', description: 'Turns AirDrop off.' };
    let response = await requestNameAndDescription({ platform: 'apple', forFleet: true, naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 200);
    assert.deepStrictEqual(response.body, { name: 'Disable AirDrop', description: 'Turns AirDrop off.' });
  });

  it('is a 400 decline when the model refuses to name and describe a profile', async function() {
    stubbedOutcome = ()=>{ throw { exit: 'refused' }; };
    let response = await requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 400);
    assert.strictEqual(response.body.error, 'couldNotNameProfile');
    assert.ok(response.body.reason);
  });

  it('is a 500 when the name or description comes back blank', async function() {
    for (let blank of [{ name: 'Disable AirDrop', description: '   ' }, { name: ' ', description: 'Turns AirDrop off.' }]) {
      stubbedOutcome = blank;
      let response = await requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
      assert.strictEqual(response.status, 500);
    }
  });

  it('is a 500 when naming and describing takes too long', async function() {
    this.timeout(15000);
    stubbedOutcome = new Promise(()=>{});
    let startedAt = Date.now();
    let response = await requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' });
    assert.strictEqual(response.status, 500);
    assert.ok(Date.now() - startedAt < 9000, `answered after ${Date.now() - startedAt}ms`);
  });

  it('cuts a name and description to Fleet\'s limits, counted in code points', async function() {
    stubbedOutcome = { name: '😀'.repeat(300), description: '😀'.repeat(1100) };
    let response = await requestNameAndDescription({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' });
    assert.strictEqual(response.status, 200);
    assert.strictEqual(Array.from(response.body.name).length, 255);
    assert.strictEqual(Array.from(response.body.description).length, 1023);
  });

  it('is a 400 when instructions for a name and description are too long', async function() {
    let response = await requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'a'.repeat(2001) });
    assert.strictEqual(response.status, 400);
    assert.strictEqual(response.body.error, 'invalidInput');
    assert.strictEqual(stubCalls.length, 0);
  });

});
