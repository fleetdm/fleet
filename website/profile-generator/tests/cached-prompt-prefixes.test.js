/**
 * Every system prompt the profile generator caches must be byte-for-byte the same on every request: a
 * prompt that differs by one character is a cache miss that still pays for the cache write.  Nothing about
 * that shows up in a generated profile, so it is checked here, without calling a model.
 */
const assert = require('assert');


// Different enough to exercise every per-request branch of the prompts: an app with an ADMX template, a
// request for a default, plain requests for each platform.
const INSTRUCTIONS = [
  'Turn off AirDrop',
  'Require a passcode with at least 8 characters',
  'Block write access to removable storage',
  'Set the Google Chrome homepage to https://intranet.example.com',
  'Set Google Chrome\'s default search engine, but let users change it',
  'Turn off the camera',
];

// Each is passed to the configuration helper alongside the instructions.  .mobileconfig and Windows prompts are
// checked both ways, since asking for settings rather than the document changes their rules and response shape.
const VARIANTS = [
  { profileType: 'mobileconfig' },
  { profileType: 'mobileconfig', writeProfilesFromSettings: true },
  { profileType: 'ddm' },
  { profileType: 'csp' },
  { profileType: 'csp', writeProfilesFromSettings: true },
  { profileType: 'android' },
];


describe('cached prompt prefixes', function() {

  let originalPromptHelper;
  let promptCalls = [];
  let appleLookupFindsNothing = false;

  before(function() {
    originalPromptHelper = sails.helpers.ai.prompt;
    // Answers each lookup with something that exists, so the rest of the configuration helper runs the way
    // it does in production.
    sails.helpers.ai.prompt = {
      with(inputs) {
        promptCalls.push(inputs);
        let answer = /Windows CSP node a profile can set/.test(inputs.systemPrompt || '') ? { areas: ['Storage'] }
          : /payload type Apple publishes/.test(inputs.systemPrompt || '') ? { payloadTypes: appleLookupFindsNothing ? [] : ['com.apple.applicationaccess'] }
          : {};
        let deferred = Promise.resolve(answer);
        deferred.tolerate = ()=>{ return deferred; };
        deferred.intercept = ()=>{ return deferred; };
        return deferred;
      },
    };
  });

  after(function() {
    sails.helpers.ai.prompt = originalPromptHelper;
  });

  for (let variant of VARIANTS) {
    let options = Object.keys(variant).filter((key)=>{ return key !== 'profileType'; }).map((key)=>{ return `${key}: ${variant[key]}`; });
    // A .mobileconfig whose lookup found nothing is generated from the whole schema, and that is the only
    // .mobileconfig generation prompt that is cached.
    for (let findsNothing of variant.profileType === 'mobileconfig' ? [false, true] : [false]) {
      let conditions = options.concat(findsNothing ? ['when the Apple lookup finds nothing'] : []);

      it(`caches only ${variant.profileType} system prompts that are the same for every request${conditions.length > 0 ? ` (${conditions.join(', ')})` : ''}`, async function() {
        appleLookupFindsNothing = findsNothing;
        let cachedSystemPromptsByModel = {};
        let generationSystemPrompts = [];
        for (let instruction of INSTRUCTIONS) {
          promptCalls = [];
          let configuration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with(Object.assign({
            naturalLanguageInstructions: instruction,
            useLighterResponseShape: true,
            useApplePayloadTypeLookup: true,
          }, variant));
          for (let call of promptCalls) {
            if(call.systemPromptCacheTtl) {
              let key = `${call.baseModel} lookup`;
              cachedSystemPromptsByModel[key] = (cachedSystemPromptsByModel[key] || []).concat([call.systemPrompt]);
            }
          }
          if(configuration.systemPromptIsTheSameForEveryRequest) {
            generationSystemPrompts.push(configuration.systemPrompt);
          }
        }

        if(variant.profileType === 'mobileconfig' || variant.profileType === 'csp') {
          assert.ok(Object.keys(cachedSystemPromptsByModel).length > 0, `the ${variant.profileType} lookup's system prompt should be cached`);
        }
        for (let [key, systemPrompts] of Object.entries(cachedSystemPromptsByModel)) {
          assert.strictEqual(_.uniq(systemPrompts).length, 1, `the ${variant.profileType} ${key}'s cached system prompt differs between requests`);
        }
        assert.ok(_.uniq(generationSystemPrompts).length <= 1, `a ${variant.profileType} generation system prompt marked as the same for every request differs between requests`);
        if(variant.profileType === 'ddm' || variant.profileType === 'android' || findsNothing) {
          assert.strictEqual(generationSystemPrompts.length, INSTRUCTIONS.length, `every ${variant.profileType} generation system prompt should be cached`);
        }
      });
    }
  }

});
