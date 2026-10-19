/**
 * What Fleet's server can rely on from the profile generator's endpoints over HTTP, checked without calling a
 * model: the generation and the model are stubbed, so what is tested is how the actions report each outcome.
 * Sails turns a thrown exit, e.g. {invalidInput: ...}, into that exit's response, and anything else into a 500.
 */
const assert = require('assert');
const path = require('path');


describe('profile generator endpoints over HTTP', function() {

  let profileAction;
  let nameAndDescriptionAction;
  let originalGenerationHelper;
  let originalPromptHelper;
  // What the stubbed generation or model call does: a result, or a function that throws.
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
    profileAction = require(path.resolve(sails.config.appPath, 'api/controllers/get-llm-generated-configuration-profile.js'));
    nameAndDescriptionAction = require(path.resolve(sails.config.appPath, 'api/controllers/get-llm-generated-configuration-profile-name-and-description.js'));
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

  // The action's own fn, as Sails calls it for a plain HTTP request, with the inputs' defaults applied.
  let requestProfile = (inputs)=>{
    return profileAction.fn.call({ req: { isSocket: false } }, _.extend({ forFleet: true }, inputs));
  };
  let requestNameAndDescription = (inputs)=>{
    return nameAndDescriptionAction.fn.call({ req: { isSocket: false } }, inputs);
  };
  let thrownBy = async (promise)=>{
    try {
      await promise;
    } catch (thrown) {
      return thrown;
    }
    assert.fail('expected the request to fail');
  };
  let generatedResult = { profileType: 'ddm', couldNotGenerateProfile: false, generatedProfile: { profile: '{}', profileFilename: 'x.json', deliveryNotes: '', items: [], additionalProfiles: [] } };

  it('is a 400 when neither or both of profileType and platform are given', async function() {
    for (let inputs of [{}, { profileType: 'ddm', platform: 'apple' }]) {
      let thrown = await thrownBy(requestProfile(_.extend({ naturalLanguageInstructions: 'Turn off AirDrop' }, inputs)));
      assert.strictEqual(thrown.invalidInput.error, 'invalidInput');
    }
    assert.strictEqual(stubCalls.length, 0);
  });

  it('counts instructions in code points: 2,000 are accepted and 2,001 are a 400', async function() {
    stubbedOutcome = generatedResult;
    let accepted = await requestProfile({ platform: 'apple', naturalLanguageInstructions: '😀'.repeat(2000) });
    assert.strictEqual(accepted.profileType, 'ddm');
    let thrown = await thrownBy(requestProfile({ platform: 'apple', naturalLanguageInstructions: '😀'.repeat(2001) }));
    assert.strictEqual(thrown.invalidInput.error, 'invalidInput');
    assert.strictEqual(stubCalls.length, 1);
  });

  it('names the profile type first in a generated profile', async function() {
    stubbedOutcome = generatedResult;
    let response = await requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Require a passcode' });
    assert.strictEqual(Object.keys(response)[0], 'profileType');
    assert.strictEqual(response.profileFilename, 'x.json');
  });

  it('is a 400 with the reason when the model declines', async function() {
    stubbedOutcome = { couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: 'This request needs two profiles.' };
    let thrown = await thrownBy(requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' }));
    assert.deepStrictEqual(thrown.couldNotGenerateProfile, { error: 'couldNotGenerateProfile', reason: 'This request needs two profiles.' });

    stubbedOutcome = { couldNotGenerateProfile: true };
    thrown = await thrownBy(requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' }));
    assert.ok(thrown.couldNotGenerateProfile.reason, 'a decline with no reason from the model still gets one');
  });

  it('is a 500, not a decline, when generation fails or produces the wrong kind of profile', async function() {
    stubbedOutcome = ()=>{ throw new Error('overloaded'); };
    let thrown = await thrownBy(requestProfile({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' }));
    assert.ok(_.isError(thrown) && !thrown.couldNotGenerateProfile, `threw ${require('util').inspect(thrown)}`);

    stubbedOutcome = { profileType: 'mobileconfig', couldNotGenerateProfile: true, contentDidNotMatchProfileType: true };
    thrown = await thrownBy(requestProfile({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' }));
    assert.ok(_.isError(thrown) && !thrown.couldNotGenerateProfile, `threw ${require('util').inspect(thrown)}`);
  });

  it('is a 400 decline when the model refuses to name and describe a profile', async function() {
    stubbedOutcome = ()=>{ throw { exit: 'refused' }; };
    let thrown = await thrownBy(requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'Turn off AirDrop' }));
    assert.strictEqual(thrown.couldNotNameProfile.error, 'couldNotNameProfile');
    assert.ok(thrown.couldNotNameProfile.reason);
  });

  it('cuts a name and description to Fleet\'s limits, counted in code points', async function() {
    stubbedOutcome = { name: '😀'.repeat(300), description: '😀'.repeat(1100) };
    let response = await requestNameAndDescription({ platform: 'windows', naturalLanguageInstructions: 'Block removable storage' });
    assert.strictEqual(Array.from(response.name).length, 255);
    assert.strictEqual(Array.from(response.description).length, 1023);
  });

  it('is a 400 when instructions for a name and description are too long', async function() {
    let thrown = await thrownBy(requestNameAndDescription({ platform: 'apple', naturalLanguageInstructions: 'a'.repeat(2001) }));
    assert.strictEqual(thrown.invalidInput.error, 'invalidInput');
    assert.strictEqual(stubCalls.length, 0);
  });

});
