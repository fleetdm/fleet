/**
 * What the generation pipeline does with a model's answer -- whether it is the kind of profile that was asked for,
 * and when a failed call is retried -- checked without calling a model: every call to the prompt helper is stubbed.
 */
const assert = require('assert');


describe('configuration profile generation pipeline', function() {

  let originalPromptHelper;
  let originalNow;
  // What each generation call does, in order: a response, an error to throw, or `refused`.
  let generationOutcomes = [];
  let generationCalls = 0;

  before(function() {
    originalPromptHelper = sails.helpers.ai.prompt;
    originalNow = Date.now;
    sails.helpers.ai.prompt = {
      with(inputs) {
        // Told apart by the user prompt: a .mobileconfig generation's system prompt can carry the same schema
        // text as the Apple lookup's.
        let isGeneration = /^Given these instructions from an IT admin, generate/.test(inputs.prompt || '');
        let outcome;
        if(isGeneration) {
          generationCalls++;
          outcome = generationOutcomes.shift();
        } else if(/Windows CSP node a profile can set/.test(inputs.systemPrompt || '')) {
          outcome = { areas: ['Storage'] };
        } else {
          outcome = { payloadTypes: ['com.apple.applicationaccess'] };
        }
        let settled = Promise.resolve().then(()=>{
          if(_.isFunction(outcome)) {
            return outcome();
          }
          if(outcome === 'refused') {
            throw { exit: 'refused' };
          }
          if(_.isError(outcome)) {
            throw outcome;
          }
          return outcome;
        });
        // Like a machine's: `.tolerate(exitName, handler)` handles only that exit, and a bare handler every error.
        settled.tolerate = (exitName, handler)=>{
          if(_.isFunction(exitName)) {
            return settled.catch(exitName);
          }
          return settled.catch((err)=>{
            if(err && err.exit === exitName) {
              return handler(err);
            }
            throw err;
          });
        };
        settled.intercept = ()=>{ return settled; };
        return settled;
      },
    };
  });

  beforeEach(function() {
    generationOutcomes = [];
    generationCalls = 0;
  });

  afterEach(function() {
    Date.now = originalNow;
  });

  after(function() {
    sails.helpers.ai.prompt = originalPromptHelper;
  });

  let generate = async (profileType, forFleet = false, naturalLanguageInstructions = 'Turn off the camera')=>{
    return await sails.helpers.generateConfigurationProfile.with({ profileType, naturalLanguageInstructions, useApplePayloadTypeLookup: false, forFleet });
  };
  let responseWith = (configurationProfile)=>{
    return { configurationProfile, profileFilename: 'profile', deliveryNotes: '', settingsEnforced: [{ name: 'setting', value: 'value' }] };
  };
  let apiError = (statusCode)=>{
    return _.extend(new Error(`The LLM API responded with a ${statusCode} status code`), { statusCode });
  };
  let declaration = JSON.stringify({ Type: 'com.apple.configuration.passcode.settings', Identifier: 'passcode', Payload: { MinimumLength: 8 } });

  describe('writes a .mobileconfig from the payload settings the model returns', function() {

    it('fails a generation whose settings cannot be written, even when the model also wrote a document of its own', async function() {
      let ownDocument = { configurationProfile: '<?xml version="1.0"?><plist version="1.0"><dict/></plist>', settingsEnforced: [{ name: 'allowCamera', value: 'disabled' }] };
      generationOutcomes = [_.extend({ profileFilename: 'camera.mobileconfig', deliveryNotes: '', payloads: [{ payloadType: 'com.apple.applicationaccess', settings: { allowCamera: 'disabled' } }] }, ownDocument)];
      let result = await generate('mobileconfig');
      assert.strictEqual(result.couldNotGenerateProfile, true);
      assert.ok(/allowCamera takes true or false/.test(result.reasonWhyAProfileCouldNotBeGenerated), result.reasonWhyAProfileCouldNotBeGenerated);
      generationOutcomes = [{ profileFilename: 'scan.xml', deliveryNotes: '', commands: [{ verb: 'Exec', locURI: './Device/Vendor/MSFT/Defender/Scan/Quick' }], configurationProfile: '<Exec><Item><Target><LocURI>./Device/Vendor/MSFT/Defender/Scan/Quick</LocURI></Target></Item></Exec>', settingsEnforced: [{ name: 'Scan/Quick', value: '' }] }];
      let windows = await generate('csp');
      assert.strictEqual(windows.couldNotGenerateProfile, true);
      assert.ok(/runs a one-time command/.test(windows.reasonWhyAProfileCouldNotBeGenerated), windows.reasonWhyAProfileCouldNotBeGenerated);
    });

    it('ends the root PayloadIdentifier with a segment that differs between two generations of the same request', async function() {
      let settingsResponse = { profileFilename: 'camera.mobileconfig', profileDisplayName: 'Camera Disabled', deliveryNotes: '', payloads: [{ payloadType: 'com.apple.applicationaccess', settings: { allowCamera: false } }] };
      generationOutcomes = [settingsResponse, settingsResponse];
      let rootIdentifiers = [];
      for (let generation of [await generate('mobileconfig'), await generate('mobileconfig')]) {
        assert.strictEqual(generation.couldNotGenerateProfile, false);
        let identifiers = Array.from(generation.generatedProfile.profile.matchAll(/<key>PayloadIdentifier<\/key>\s*<string>([^<]*)<\/string>/g), (match)=>{ return match[1]; });
        let root = _.find(identifiers, (identifier)=>{ return /^com\.example\.camera\.[0-9a-f]{8}$/.test(identifier); });
        assert.ok(root, identifiers.join(', '));
        rootIdentifiers.push(root);
      }
      assert.notStrictEqual(rootIdentifiers[0], rootIdentifiers[1]);
    });

  });

  describe('takes a platform from Fleet in place of a profile type', function() {

    it('generates a Windows CSP profile for the windows platform', async function() {
      generationOutcomes = [{ profileFilename: 'camera.xml', deliveryNotes: '', commands: [{ verb: 'Replace', locURI: './Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera', data: 0 }] }];
      let result = await sails.helpers.generateConfigurationProfile.with({ platform: 'windows', forFleet: true, naturalLanguageInstructions: 'Turn off the camera' });
      assert.strictEqual(result.couldNotGenerateProfile, false, result.reasonWhyAProfileCouldNotBeGenerated);
      assert.strictEqual(result.profileType, 'csp');
      assert.deepStrictEqual(result.windowsCspAreasProvided, ['Storage']);
      assert.ok(/<LocURI>\.\/Device\/Vendor\/MSFT\/Policy\/Config\/Camera\/AllowCamera<\/LocURI>/.test(result.generatedProfile.profile), result.generatedProfile.profile);
    });

  });

  describe('checks that the content is the kind of profile that was asked for', function() {

    it('rejects prose as a Windows profile', async function() {
      generationOutcomes = [responseWith('Set Camera/AllowCamera to 0 to turn the camera off.')];
      let result = await generate('csp');
      assert.strictEqual(result.couldNotGenerateProfile, true);
      assert.strictEqual(result.contentDidNotMatchProfileType, true);
    });

    it('accepts a Windows profile that opens with a command', async function() {
      generationOutcomes = [responseWith('<Replace>\n  <Item>\n    <Target>\n      <LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI>\n    </Target>\n    <Data>0</Data>\n  </Item>\n</Replace>')];
      let result = await generate('csp');
      assert.strictEqual(result.couldNotGenerateProfile, false);
    });

    it('rejects a declaration as an Android policy', async function() {
      generationOutcomes = [responseWith(declaration)];
      let result = await generate('android');
      assert.strictEqual(result.contentDidNotMatchProfileType, true);
    });

    it('accepts an Android policy', async function() {
      generationOutcomes = [responseWith('{"cameraDisabled": true}')];
      let result = await generate('android');
      assert.strictEqual(result.couldNotGenerateProfile, false);
    });

    it('accepts a property list as a .mobileconfig', async function() {
      generationOutcomes = [responseWith('<?xml version="1.0" encoding="UTF-8"?>\n<plist version="1.0">\n<dict>\n</dict>\n</plist>')];
      let result = await generate('mobileconfig');
      assert.strictEqual(result.couldNotGenerateProfile, false);
    });

    it('accepts a declaration as a DDM profile', async function() {
      generationOutcomes = [responseWith(declaration)];
      let result = await generate('ddm');
      assert.strictEqual(result.couldNotGenerateProfile, false);
    });

  });

  describe('the declaration schema Fleet is given', function() {

    it('has no asset references at any depth, and keeps the rest of a type that only nests them', async function() {
      let configuration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({ profileType: 'ddm', naturalLanguageInstructions: 'Configure a managed app', forFleet: true });
      // Only the rule that forbids them names them.
      assert.doesNotMatch(configuration.systemPrompt.replace(/Never write a key that ends in AssetReference[^\n]*/, ''), /AssetReference/);
      assert.match(configuration.systemPrompt, /com\.apple\.configuration\.app\.managed/);
      assert.doesNotMatch(configuration.systemPrompt, /com\.apple\.configuration\.security\.certificate\b/);
    });

  });

  describe('keeps links out of a decline reason', function() {

    for (let [reason, expected] of [
      ['Edge has no CSP nodes. See https://learn.microsoft.com/en-us/deployedge/ for its ADMX templates.', 'Edge has no CSP nodes. See for its ADMX templates.'],
      ['Use the [Edge ADMX templates](https://learn.microsoft.com/deployedge) instead.', 'Use the Edge ADMX templates instead.'],
      ['No such setting (www.example.com).', 'No such setting.'],
      ['No such setting.', 'No such setting.'],
    ]) {
      it(JSON.stringify(reason), async function() {
        generationOutcomes = [{ couldNotGenerateProfile: true, reasonWhyAProfileCouldNotBeGenerated: reason }];
        let result = await generate('csp');
        assert.strictEqual(result.couldNotGenerateProfile, true);
        assert.strictEqual(result.reasonWhyAProfileCouldNotBeGenerated, expected);
      });
    }

  });

  describe('scopes a user-channel-only declaration to the user', function() {

    let calDav = JSON.stringify({ Type: 'com.apple.configuration.account.caldav', Identifier: 'caldav', Payload: { HostName: 'cal.acme.example' } });

    it('when Fleet asks', async function() {
      generationOutcomes = [responseWith(calDav)];
      let result = await generate('ddm', true);
      assert.strictEqual(JSON.parse(result.generatedProfile.profile).PayloadScope, 'User');
      assert.match(result.generatedProfile.deliveryNotes, /label/);
      assert.match(result.generatedProfile.deliveryNotes, /user channel/);
    });

    it('in place of another scope, which would deliver it to the device channel', async function() {
      generationOutcomes = [responseWith(JSON.stringify(Object.assign(JSON.parse(calDav), { PayloadScope: 'System' })))];
      let result = await generate('ddm', true);
      assert.strictEqual(JSON.parse(result.generatedProfile.profile).PayloadScope, 'User');
    });

    it('but not when the request names only iPhones or iPads, where Fleet fails a User-scoped declaration', async function() {
      generationOutcomes = [responseWith(calDav)];
      let result = await generate('ddm', true, 'Add our CardDAV server cal.acme.example on iPhones');
      assert.strictEqual(JSON.parse(result.generatedProfile.profile).PayloadScope, undefined);
      assert.doesNotMatch(result.generatedProfile.deliveryNotes || '', /label/);
    });

    it('but not otherwise', async function() {
      generationOutcomes = [responseWith(calDav)];
      let result = await generate('ddm');
      assert.strictEqual(JSON.parse(result.generatedProfile.profile).PayloadScope, undefined);
    });

  });

  describe('retries a failed generation once', function() {

    for (let statusCode of [429, 529, 500]) {
      it(`after a ${statusCode}`, async function() {
        generationOutcomes = [apiError(statusCode), responseWith(declaration)];
        let result = await generate('ddm');
        assert.strictEqual(generationCalls, 2);
        assert.strictEqual(result.couldNotGenerateProfile, false);
      });
    }

    it('but not after an error in the request itself', async function() {
      generationOutcomes = [apiError(400), responseWith(declaration)];
      await assert.rejects(generate('ddm'), (err)=>{ return /400/.test(err.message); });
      assert.strictEqual(generationCalls, 1);
    });

    it('but not after a refusal, which is a decline', async function() {
      generationOutcomes = ['refused', responseWith(declaration)];
      let result = await generate('ddm');
      assert.strictEqual(generationCalls, 1);
      assert.strictEqual(result.couldNotGenerateProfile, true);
    });

    it('but not once too little of the latency budget is left for it', async function() {
      // The first call fails after the clock has moved on far enough that a retry would finish too late.
      generationOutcomes = [()=>{
        let startedLateAt = originalNow() + 5000;
        Date.now = ()=>{ return startedLateAt; };
        throw apiError(529);
      }, responseWith(declaration)];
      await assert.rejects(generate('ddm'), (err)=>{ return /529/.test(err.message); });
      assert.strictEqual(generationCalls, 1);
    });

  });

});


describe('reading a model\'s JSON answer', function() {

  let originalHttpPost;
  let originalSecret;
  let replyText;

  before(function() {
    originalHttpPost = sails.helpers.http.post;
    originalSecret = sails.config.custom.anthropicSecret;
    sails.config.custom.anthropicSecret = 'not-a-real-key';
    // Like the HTTP helper's deferred: `.intercept()` chains, and awaiting it gives the API's response body.
    sails.helpers.http.post = ()=>{
      let response = Promise.resolve({ 'stop_reason': 'end_turn', content: [{ type: 'text', text: replyText }] });
      response.intercept = ()=>{ return response; };
      return response;
    };
  });

  after(function() {
    sails.helpers.http.post = originalHttpPost;
    sails.config.custom.anthropicSecret = originalSecret;
  });

  let answerTo = async (text)=>{
    replyText = text;
    return await sails.helpers.ai.prompt.with({ prompt: 'Generate something.', baseModel: 'claude-sonnet-5-5', expectJson: true });
  };

  it('reads a plain JSON answer', async function() {
    assert.deepStrictEqual(await answerTo('{"profileFilename": "camera.mobileconfig"}'), { profileFilename: 'camera.mobileconfig' });
  });

  it('reads a JSON answer in a code fence', async function() {
    assert.deepStrictEqual(await answerTo('```json\n{"a": 1}\n```'), { a: 1 });
  });

  it('reads the JSON when a note follows it', async function() {
    assert.deepStrictEqual(await answerTo('{"a": "}"}\n\nNote: the brace in the value is text.'), { a: '}' });
  });

  it('reads the corrected answer when a second one follows the first', async function() {
    assert.deepStrictEqual(await answerTo('{"a": 1}\nCorrected output:\n{"a": 2}'), { a: 2 });
  });

  it('fails when there is no JSON object at all', async function() {
    await assert.rejects(answerTo('I cannot help with that.'), /Expecting JSON result from LLM/);
  });

});
