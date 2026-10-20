/**
 * How an Apple request is routed, and the lifecycle notes an Apple profile gets, checked without calling a model:
 * the router's reply is stubbed, so what is tested is everything the router does with it.
 */
const assert = require('assert');


// Settings as the router would map them, using keys that exist in the pinned schemas.
const MINIMUM_PASSCODE_LENGTH = { setting: 'minimum passcode length', declaration: 'com.apple.configuration.passcode.settings/MinimumLength', payload: 'com.apple.mobiledevice.passwordpolicy/minLength' };
const AIRDROP = { setting: 'AirDrop', declaration: null, payload: 'com.apple.applicationaccess/allowAirDrop' };
const INSTALL_MACOS_UPDATES = { setting: 'install macOS updates automatically', declaration: 'com.apple.configuration.softwareupdate.settings/AutomaticActions.InstallOSUpdates', payload: 'com.apple.SoftwareUpdate/AutomaticallyInstallMacOSUpdates' };
const CALCULATOR_SCIENTIFIC_MODE = { setting: 'calculator scientific mode', declaration: 'com.apple.configuration.math.settings/Calculator.ScientificMode', payload: null };
const INSTALL_PROFILE_FROM_URL = { setting: 'install a profile from a URL', declaration: 'com.apple.configuration.legacy/ProfileURL', payload: null };
const AUTO_LOGOUT_DELAY = { setting: 'log out after inactivity', declaration: null, payload: '.GlobalPreferences/com.apple.autologout.AutoLogOutDelay' };
const CONFIGURATION_FILE_SERVICE = { setting: 'configure a system service', declaration: 'com.apple.configuration.services.configuration-files/ServiceType', payload: null };


describe('Apple request routing', function() {

  let originalPromptHelper;
  // What the stubbed router call does: a reply, a function that throws, or a promise that never settles.
  let routerReply;

  before(function() {
    originalPromptHelper = sails.helpers.ai.prompt;
    sails.helpers.ai.prompt = {
      with() {
        let outcome = Promise.resolve().then(()=>{ return _.isFunction(routerReply) ? routerReply() : routerReply; });
        outcome.tolerate = (handler)=>{ return outcome.catch(handler); };
        return outcome;
      },
    };
  });

  after(function() {
    sails.helpers.ai.prompt = originalPromptHelper;
  });

  let routeWith = async (naturalLanguageInstructions, reply)=>{
    routerReply = reply;
    return await sails.helpers.routeAppleConfigurationProfileRequest.with({ naturalLanguageInstructions, forFleet: true });
  };
  let replyMapping = (settings, payloadTypes)=>{ return JSON.stringify({ settings, payloadTypes: payloadTypes || [] }); };

  it('routes to a declaration when one declaration type covers every setting', async function() {
    let route = await routeWith('Require a passcode with at least 8 characters', replyMapping([MINIMUM_PASSCODE_LENGTH]));
    assert.strictEqual(route.profileType, 'ddm');
    assert.deepStrictEqual(route.mobileconfigAlternative, ['com.apple.mobiledevice.passwordpolicy/minLength']);
  });

  it('routes to a .mobileconfig when a setting has no declaration', async function() {
    let route = await routeWith('Require an 8-character passcode and turn off AirDrop', replyMapping([MINIMUM_PASSCODE_LENGTH, AIRDROP]));
    assert.strictEqual(route.profileType, 'mobileconfig');
    assert.ok(_.contains(route.applePayloadTypes, 'com.apple.applicationaccess'));
    assert.ok(_.contains(route.applePayloadTypes, 'com.apple.mobiledevice.passwordpolicy'));
  });

  it('declines when a setting a declaration covers would need a removed .mobileconfig key, naming the profiles needed', async function() {
    let route = await routeWith('On Macs, install macOS updates automatically and turn off AirDrop', replyMapping([INSTALL_MACOS_UPDATES, AIRDROP]));
    assert.strictEqual(route.couldNotGenerateProfile, true);
    assert.ok(/^This request needs two profiles\./.test(route.reasonWhyAProfileCouldNotBeGenerated), route.reasonWhyAProfileCouldNotBeGenerated);
    assert.ok(_.contains(route.reasonWhyAProfileCouldNotBeGenerated, '"AirDrop" needs a .mobileconfig.'), route.reasonWhyAProfileCouldNotBeGenerated);
  });

  it('puts a declaration-only setting in its own declaration when declining, and counts that profile too', async function() {
    let route = await routeWith('On Macs, install macOS updates automatically, turn off AirDrop, and allow the calculator\'s scientific mode', replyMapping([INSTALL_MACOS_UPDATES, AIRDROP, CALCULATOR_SCIENTIFIC_MODE]));
    let reason = route.reasonWhyAProfileCouldNotBeGenerated;
    assert.ok(/^This request needs three profiles\./.test(reason), reason);
    assert.ok(_.contains(reason, '"calculator scientific mode" needs the com.apple.configuration.math.settings declaration.'), reason);
    assert.ok(!/"calculator scientific mode"[^.]*\.mobileconfig/.test(reason), reason);
  });

  it('declines rather than dropping a setting only a declaration can enforce from a .mobileconfig', async function() {
    let route = await routeWith('Turn off AirDrop and allow the calculator\'s scientific mode', replyMapping([AIRDROP, CALCULATOR_SCIENTIFIC_MODE]));
    let reason = route.reasonWhyAProfileCouldNotBeGenerated;
    assert.strictEqual(route.couldNotGenerateProfile, true);
    assert.ok(/^This request needs two profiles\./.test(reason), reason);
    assert.ok(_.contains(reason, '"calculator scientific mode" needs the com.apple.configuration.math.settings declaration.'), reason);
    assert.ok(_.contains(reason, '"AirDrop" needs a .mobileconfig.'), reason);
  });

  it('keeps a key whose name has dots in it, rather than reading the dots as nesting', async function() {
    let lifecycle = sails.helpers.getAppleSchemaLifecycle.with({ profileType: 'mobileconfig', name: '.GlobalPreferences', keyPath: 'com.apple.autologout.AutoLogOutDelay' });
    assert.strictEqual(lifecycle.found, true);
    assert.strictEqual(lifecycle.keyPath, 'com.apple.autologout.AutoLogOutDelay');
    let route = await routeWith('Log people out after 10 minutes of inactivity', replyMapping([AUTO_LOGOUT_DELAY]));
    assert.strictEqual(route.profileType, 'mobileconfig');
    assert.ok(route.settings[0].payload, 'the mapping was dropped');
    assert.ok(_.contains(route.applePayloadTypes, '.GlobalPreferences'));
  });

  it('does not use a declaration type on its own when it requires an asset, even for a key that is not the asset', async function() {
    let route = await routeWith('Configure the SSH service', replyMapping([CONFIGURATION_FILE_SERVICE]));
    assert.notStrictEqual(route.profileType, 'ddm');
  });

  for (let [description, reply] of [
    ['the router fails', ()=>{ throw new Error('overloaded'); }],
    ['the router\'s reply is not JSON', 'I could not work this out.'],
  ]) {
    it(`falls back to a .mobileconfig from the full schema when ${description}`, async function() {
      let route = await routeWith('Turn off AirDrop', reply);
      assert.strictEqual(route.profileType, 'mobileconfig');
      assert.deepStrictEqual(route.applePayloadTypes, []);
    });
  }

  it('gives up on a router that takes too long, and falls back to a .mobileconfig from the full schema', async function() {
    this.timeout(15000);
    let route = await routeWith('Turn off AirDrop', new Promise(()=>{}));
    assert.strictEqual(route.profileType, 'mobileconfig');
    assert.deepStrictEqual(route.applePayloadTypes, []);
    assert.ok(route.routerMs >= 5000 && route.routerMs < 6000, `routed in ${route.routerMs}ms`);
  });

  it('reads the last JSON object when the router corrects itself', async function() {
    let reply = `${replyMapping([AIRDROP])}\nCorrected output:\n${replyMapping([MINIMUM_PASSCODE_LENGTH])}`;
    let route = await routeWith('Require a passcode with at least 8 characters', reply);
    assert.strictEqual(route.profileType, 'ddm');
  });

  it('drops paths that do not exist, including ones with an empty key', async function() {
    let route = await routeWith('Require a passcode with at least 8 characters', replyMapping([
      _.extend({}, MINIMUM_PASSCODE_LENGTH, { declaration: 'com.apple.configuration.passcode.settings/NotARealKey' }),
      { setting: 'something else', declaration: 'com.apple.configuration.passcode.settings/', payload: null },
    ]));
    assert.strictEqual(route.settings[0].declaration, undefined);
    assert.strictEqual(route.settings[1].declaration, undefined);
    assert.strictEqual(route.profileType, 'mobileconfig');
  });

  it('rewrites a path written in the wrong case in the schema\'s casing', async function() {
    let route = await routeWith('Require a passcode with at least 8 characters', replyMapping([
      { setting: 'minimum passcode length', declaration: 'COM.APPLE.configuration.passcode.settings/minimumlength', payload: 'com.apple.mobiledevice.passwordpolicy/MINLENGTH' },
    ]));
    assert.strictEqual(route.profileType, 'ddm');
    assert.strictEqual(route.settings[0].declaration.typeName, 'com.apple.configuration.passcode.settings');
    assert.strictEqual(route.settings[0].declaration.keyPath, 'MinimumLength');
    assert.deepStrictEqual(route.mobileconfigAlternative, ['com.apple.mobiledevice.passwordpolicy/minLength']);
  });

  it('uses the legacy declaration only when the instructions point at a .mobileconfig file', async function() {
    assert.strictEqual((await routeWith('Install the profile at https://example.com/wifi.mobileconfig', replyMapping([INSTALL_PROFILE_FROM_URL]))).profileType, 'ddm');
    assert.strictEqual((await routeWith('Install our Wi-Fi profile', replyMapping([INSTALL_PROFILE_FROM_URL]))).profileType, 'mobileconfig');
  });

});


describe('Apple lifecycle notes', function() {

  let mobileconfigWith = (payloadType, keyNames)=>{
    return `<plist version="1.0"><dict><key>PayloadContent</key><array><dict><key>PayloadType</key><string>${payloadType}</string>` +
      _.map(keyNames, (keyName)=>{ return `<key>${keyName}</key><true/>`; }).join('') +
      '</dict></array><key>PayloadType</key><string>Configuration</string></dict></plist>';
  };
  let notesFor = (inputs)=>{ return sails.helpers.addLifecycleNotesToAppleProfile.with(inputs); };

  it('names every platform that deprecates a key in the same version in one sentence', function() {
    let notes = notesFor({ profileType: 'mobileconfig', profile: mobileconfigWith('com.apple.applicationaccess', ['allowWritingTools']) });
    assert.deepStrictEqual(notes, ['Apple deprecated allowWritingTools in iOS, macOS, and visionOS 26.4; it still works for now.']);
  });

  it('reports only the platforms the instructions name', function() {
    let notes = notesFor({ profileType: 'mobileconfig', profile: mobileconfigWith('com.apple.applicationaccess', ['allowWritingTools']), platformsNamed: ['macOS'] });
    assert.deepStrictEqual(notes, ['Apple deprecated allowWritingTools in macOS 26.4; it still works for now.']);
  });

  it('says when a payload type stops working, without repeating it for each of its keys', function() {
    let notes = notesFor({ profileType: 'mobileconfig', profile: mobileconfigWith('com.apple.SoftwareUpdate', ['AutomaticallyInstallMacOSUpdates']) });
    assert.deepStrictEqual(notes, ['com.apple.SoftwareUpdate stops working in macOS 27.0.']);
  });

  it('looks a key up under its own payload type by its full path when the profile was written from settings', function() {
    let profile = mobileconfigWith('com.apple.extensiblesso', ['PlatformSSO', 'AuthenticationMethod']);
    // Nested under PlatformSSO, it is not the deprecated top-level AuthenticationMethod of the same name.
    let nested = notesFor({ profileType: 'mobileconfig', profile, payloads: [{ payloadType: 'com.apple.extensiblesso', settings: { PlatformSSO: { AuthenticationMethod: 'Password' } } }] });
    assert.deepStrictEqual(nested, []);
    let topLevel = notesFor({ profileType: 'mobileconfig', profile, payloads: [{ payloadType: 'com.apple.extensiblesso', settings: { AuthenticationMethod: 'Password' } }] });
    assert.deepStrictEqual(topLevel, ['Apple deprecated AuthenticationMethod in macOS 14.0; it still works for now.']);
    // A key is only looked up in the payload that sets it.
    let elsewhere = notesFor({ profileType: 'mobileconfig', profile, payloads: [
      { payloadType: 'com.apple.extensiblesso', settings: { PlatformSSO: { AuthenticationMethod: 'Password' } } },
      { payloadType: 'com.apple.applicationaccess', settings: { allowCamera: false } },
    ] });
    assert.deepStrictEqual(elsewhere, []);
  });

  it('gives a declaration a minimum OS only where the .mobileconfig it was chosen over worked on an older one', function() {
    let declaration = JSON.stringify({ Type: 'com.apple.configuration.passcode.settings', Identifier: 'com.example.passcode', Payload: { MinimumLength: 8 } });
    let notes = notesFor({ profileType: 'ddm', profile: declaration, mobileconfigAlternative: ['com.apple.mobiledevice.passwordpolicy/minLength'] });
    assert.deepStrictEqual(notes, ['This declaration needs iOS 15.0 and macOS 13.0 or later.']);
  });

});


describe('Apple platforms named in instructions', function() {

  for (let [instructions, expected] of [
    ['Turn off AirDrop on Macs and iPhones', ['macOS', 'iOS']],
    ['Require a passcode on iPadOS', ['iOS']],
    ['Turn off the screen saver on Apple TV', ['tvOS']],
    ['Block MAC address randomization', []],
    ['Turn off AirDrop', []],
  ]) {
    it(`finds ${expected.length > 0 ? expected.join(' and ') : 'no platform'} in "${instructions}"`, function() {
      assert.deepStrictEqual(sails.helpers.getApplePlatformsNamed.with({ naturalLanguageInstructions: instructions }), expected);
    });
  }

});
