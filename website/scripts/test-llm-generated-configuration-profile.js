//  ╔═╗╔═╗╔═╗╔═╗╔═╗
//  ║  ╠═╣╚═╗║╣ ╚═╗
//  ╚═╝╩ ╩╚═╝╚═╝╚═╝
//
// `mustContain` / `mustNotContain` are compared with all whitespace stripped from both sides,
// because the JSON formats have unpredictable indentation and an assertion like
// `"maximumTimeToLock": "300000"` would otherwise fail on formatting alone.
//
// Assertions are substring-only by design.  Anything cleverer would be a validator, and the design
// decision for this generator was that there isn't one -- the CANARY cases are written as exact
// substrings so this can stay dumb.  Properties that genuinely can't be checked this way (one dict
// per payload domain, distinct PayloadUUIDs, single-line CDATA) are noted on the case and stay a
// human read.
const TEST_CASES = [

  //  ╔═╗╔═╗╔═╗
  //  ║  ╚═╗╠═╝
  //  ╚═╝╚═╝╩
  {
    id: 'csp-device-password',
    profileType: 'csp',
    canary: true,
    instructions: 'Require a password to unlock the device.',
    readByEye: 'valueMeaning should say 0 means a password IS required.  If it describes 0 as disabling the password, the value is right by luck and the reasoning is wrong -- which will not hold on the next node.',
    expect: {
      mustContain: ['DeviceLock/DevicePasswordEnabled'],
      mustContainElement: [['Format', 'int'], ['Data', '0']],
      mustNotContainElement: [['Format', 'bool']],
      mustNotContain: ['<SyncML', '<?xml'],
    }
  },
  {
    id: 'csp-removable-storage',
    profileType: 'csp',
    instructions: 'Block write access to removable storage.',
    expect: {
      mustContain: ['RemovableDiskDenyWriteAccess'],
      mustContainElement: [['Format', 'int']],
      mustNotContainElement: [['Format', 'bool'], ['Format', 'chr']],
    }
  },
  {
    id: 'csp-min-password-length',
    profileType: 'csp',
    instructions: 'Require device passwords to be at least 12 characters long.',
    expect: {
      mustContain: ['MinDevicePasswordLength', 'DevicePasswordEnabled'],
      mustContainElement: [['Data', '12'],['Format', 'int']]
    }
  },
  {
    id: 'csp-failed-attempts',
    profileType: 'csp',
    instructions: 'Lock the device after 10 failed password attempts.',
    expect: {
      mustContain: ['MaxDevicePasswordFailedAttempts', 'DevicePasswordEnabled']
    }
  },
  {
    id: 'csp-wifi-wpa2-psk-with-spaces',
    profileType: 'csp',
    canary: true,
    instructions: 'Add a wifi profile for a network with the SSID "A Network" with WPA2 authentication that uses the password "aaaaaaapassword".',
    readByEye: 'The embedded <WLANProfile> must be on ONE line inside the CDATA -- no assertion can express that.  Also confirm <hex> decodes to the SSID, and that deliveryNotes names the cleartext passphrase rather than describing the profile.',
    expect: {
      mustContain: ['A%20Network', '<![CDATA[',],
      mustContainElement: [['name', 'A Network']],
      mustNotContain: ['&lt;WLANProfile', '<SyncML', 'A%20network'],
    }
  },
  {
    id: 'csp-logon-banner',
    profileType: 'csp',
    instructions: 'Show "Authorized users only" as a message on the sign-in screen.',
    expect: {
      mustContain: ['InteractiveLogon'],
      mustContainElement: [['Format', 'chr']]
    }
  },
  {
    id: 'csp-telemetry',
    profileType: 'csp',
    instructions: 'Set diagnostic data to the lowest level allowed.',
    expect: {
      mustContain: ['AllowTelemetry'],
      mustContainElement: [['Format', 'int']]
    }
  },

  //  ╔╦╗╔═╗╔╗ ╦╦  ╔═╗╔═╗╔╗╔╔═╗╦╔═╗
  //  ║║║║ ║╠╩╗║║  ║╣ ║  ║║║╠╣ ║║ ╦
  //  ╩ ╩╚═╝╚═╝╩╩═╝╚═╝╚═╝╝╚╝╚  ╩╚═╝
  {
    id: 'mobileconfig-mixed-casing',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Require a 12-character passcode with no simple passcodes, and turn on automatic checking for updates.',
    readByEye: 'Two payload dicts should be present, each with its own PayloadUUID and an identifier suffix.  Confirm the passcode keys kept their lowercase first letter in the SAME profile where AutomaticCheckEnabled kept its capital -- a model can be self-consistently wrong and still pass one of these two checks.',
    expect: {
      mustContain: ['forcePIN', 'minLength', 'allowSimple', 'AutomaticCheckEnabled'],
      mustNotContain: ['ForcePIN', 'MinLength', 'AllowSimple', 'automaticCheckEnabled']
    }
  },
  {
    id: 'mobileconfig-showfullname',
    profileType: 'mobileconfig',
    canary: true,
    instructions: 'Set the login window to show a list of users instead of name and password fields.',
    readByEye: 'SHOWFULLNAME false is what shows the user list.  Confirm valueMeaning explains that, rather than describing false as turning something off.',
    expect: {
      mustContain: ['SHOWFULLNAME', '<false/>'],
      mustNotContainElement: [['string', 'false']]
    }
  },
  {
    id: 'mobileconfig-disable-camera',
    profileType: 'mobileconfig',
    instructions: 'Disable the camera.',
    expect: {
      mustContain: ['com.apple.applicationaccess', 'allowCamera', '<false/>'],
      mustNotContainElement: [['string', 'false']],
    }
  },
  {
    id: 'mobileconfig-login-banner',
    profileType: 'mobileconfig',
    instructions: 'Show "Authorized use only" on the login window.',
    expect: {
      mustContain: ['LoginwindowText', 'Authorized use only']
    }
  },
  {
    id: 'mobileconfig-screensaver',
    profileType: 'mobileconfig',
    instructions: 'Lock the screen after 10 minutes of inactivity and require a password immediately.',
    expect: {
      mustContain: ['idleTime', 'askForPassword'],
      mustContainElement: [['integer', '600']]
    }
  },
  {
    id: 'mobileconfig-firewall',
    profileType: 'mobileconfig',
    instructions: 'Turn on the firewall in stealth mode and block all incoming connections.',
    readByEye: 'All three keys must sit in ONE dict of PayloadType com.apple.security.firewall.  Count the dicts inside PayloadContent.',
    expect: {
      mustContain: ['EnableFirewall', 'EnableStealthMode', 'com.apple.security.firewall']
    }
  },
  {
    id: 'mobileconfig-two-payloads',
    profileType: 'mobileconfig',
    instructions: 'Disable AirDrop and turn off Siri.',
    readByEye: 'Two dicts, each with a distinct uppercase PayloadUUID and an identifier that is the root identifier plus a suffix.  Duplicate UUIDs install unpredictably and nothing here can detect them.',
    expect: {
      mustContain: ['PayloadUUID', 'PayloadIdentifier']
    }
  },
  {
    id: 'mobileconfig-diagnostics',
    profileType: 'mobileconfig',
    instructions: 'Stop sending diagnostic and usage data to Apple.',
    expect: {
      mustContain: ['com.apple.SubmitDiagInfo', 'AutoSubmit']
    }
  },
  {
    id: 'mobileconfig-dock-lowercase-keys',
    profileType: 'mobileconfig',
    instructions: 'Set the Dock to auto-hide and pin it to the left side of the screen.',
    // All-lowercase keys -- fails if the model PascalCases.
    expect: {
      mustContain: ['com.apple.dock', 'autohide', 'orientation'],
      mustNotContain: ['Autohide', 'Orientation']
    }
  },

  //  ╔╦╗╔╦╗╔╦╗
  //   ║║ ║║║║║
  //  ═╩╝═╩╝╩ ╩
  {
    id: 'ddm-passcode-alphanumeric',
    profileType: 'ddm',
    canary: true,
    instructions: 'Require a 10-character alphanumeric passcode and lock the device after 10 failed attempts.',
    readByEye: 'Identifier must not be a copy of Type, and must be 64 bytes or fewer.  MaximumFailedAttempts accepts 2-11, so 10 is in range -- confirm it was not clamped or rewritten.',
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', 'MinimumLength', 'RequireAlphanumericPasscode'],
      mustNotContain: ['requirePasscode', 'forcePIN', 'minLength']
    }
  },
  {
    id: 'ddm-beta-enroll',
    profileType: 'ddm',
    instructions: 'Enroll devices in our beta program using the enrollment token BETA-TOKEN-123, shown to users as "Acme macOS Beta".',
    expect: {
      mustContain: ['com.apple.configuration.softwareupdate.settings', 'RequireProgram', 'BETA-TOKEN-123']
    }
  },
  {
    id: 'ddm-beta-block',
    profileType: 'ddm',
    instructions: 'Block enrollment in any beta program.',
    expect: {
      mustContain: ['com.apple.configuration.softwareupdate.settings', 'ProgramEnrollment', 'AlwaysOff'],
      mustNotContain: ['AlwaysOn']
    }
  },
  {
    id: 'ddm-defer-minor-updates',
    profileType: 'ddm',
    instructions: 'Defer minor updates by 30 days.',
    expect: {
      mustContain: ['softwareupdate.settings', 'Deferrals', 'MinorPeriodInDays', '30']
    }
  },
  {
    id: 'ddm-auto-install-security-responses',
    profileType: 'ddm',
    instructions: 'Automatically install security responses and system files.',
    readByEye: 'RapidSecurityResponse.Enable is the setting that matches the macOS wording here.  AutomaticActions.InstallSecurityUpdate is defensible too -- confirm whichever came back is a real key rather than an invented Automatic*Enabled one.',
    expect: {
      mustContain: ['softwareupdate.settings'],
      mustNotContain: ['AutomaticCheckEnabled', 'AutomaticInstallEnabled']
    }
  },
  {
    id: 'ddm-enforce-os-version',
    profileType: 'ddm',
    instructions: 'Enforce macOS 26.1 by December 15, 2026 at 6:00 PM local time.',
    expect: {
      mustContain: ['softwareupdate.enforcement.specific', 'TargetOSVersion', 'TargetLocalDateTime', '"26.1"']
    }
  },
  {
    id: 'ddm-intelligence-off',
    profileType: 'ddm',
    instructions: 'Turn off every Apple Intelligence feature.',
    expect: { mustContain: ['intelligence.settings', 'AllowWritingTools', 'AllowGenmoji', 'AllowImagePlayground'] }
  },
  {
    id: 'ddm-intelligence-partial',
    profileType: 'ddm',
    instructions: 'Block Writing Tools and Image Playground but leave the rest of Apple Intelligence available.',
    expect: {
      mustContain: ['intelligence.settings', 'AllowWritingTools', 'AllowImagePlayground']
    }
  },
  {
    id: 'ddm-migration-assistant',
    profileType: 'ddm',
    instructions: 'Turn on managed migration and keep the Downloads folder out of anything that gets migrated.',
    readByEye: 'ExcludedPaths entries are relative to the home directory and directory paths need a trailing slash, so "Downloads/" is right and "/Users/x/Downloads" is not.',
    expect: {
      mustContain: ['migration-assistant.settings', 'ShouldDoManagedMigration', 'ExcludedPaths', 'Downloads']
    }
  },
  {
    id: 'ddm-identifier-collision',
    profileType: 'ddm',
    instructions: 'Create two declarations under the com.acme namespace: one requiring a 10-character passcode, and one deferring minor updates by 30 days.',
    readByEye: 'The two Identifiers must differ, and each must derive from the full declaration type rather than its last component -- passcode.settings and softwareupdate.settings collapsing to one identifier is the defect this case exists for.',
    expect: {
      mustContain: ['com.apple.configuration.passcode.settings', 'com.apple.configuration.softwareupdate.settings', 'com.acme'],
      mustNotContain: ['"com.acme.settings"']
    }
  },


];


module.exports = {


  friendlyName: 'Test llm generated configuration profile',


  description: 'Generate configuration profiles and report wall-clock time plus whether the output holds up, for one instruction or for every case in this script.',


  extendedDescription:
`The prompt comes from api/helpers/get-configuration-profile-generator-configuration.js, the same
helper the public endpoint calls, so there is nothing to keep in sync -- only the model and effort
level vary here.


Examples:
  sails run test-llm-generated-configuration-profile --profileType=csp --naturalLanguageInstructions="Require a device password"
  sails run test-llm-generated-configuration-profile --all
  sails run test-llm-generated-configuration-profile --all --profileType=mobileconfig --verbose
  sails run test-llm-generated-configuration-profile --all --profileType=csp --baseModel=claude-haiku-4-5
  sails run test-llm-generated-configuration-profile --all --profileType=ddm --baseModel=claude-haiku-4-5 --validateWithContour
  sails run test-llm-generated-configuration-profile --profileType=ddm --naturalLanguageInstructions="Defer minor updates by 30 days" --parallelTests=5
  sails run test-llm-generated-configuration-profile --caseId=ddm-defer-minor-updates --parallelTests=5
  sails run test-llm-generated-configuration-profile --all --profileType=csp --parallelTests=5`,


  inputs: {

    profileType: {
      type: 'string',
      isIn: ['mobileconfig', 'csp', 'ddm'],
      description: 'Generate one profile of this type, or with --all, run only this type\'s cases.'
    },

    naturalLanguageInstructions: {
      type: 'string',
      description: 'The instructions to generate from.  Required unless --all is set.'
    },

    all: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Run every case defined at the top of this script.'
    },

    baseModel: {
      type: 'string',
      defaultsTo: 'claude-haiku-4-5',
      description: 'The model to generate with.'
    },

    effort: {
      type: 'string',
      isIn: ['low', 'medium', 'high', 'xhigh', 'max'],
      description: 'Effort level.  Omitted means the model\'s default, which is high on Sonnet 5.'
    },

    verbose: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Print the full profile and citations for every case.  Failures print them regardless.'
    },

    caseId: {
      type: 'string',
      description: 'Run one case from the list at the top of this script, by its id, e.g. "ddm-defer-minor-updates".',
      extendedDescription: `Unlike --naturalLanguageInstructions, the case brings its own expect block, so repeated runs
are actually scored rather than just printed.  Combines with --parallelTests to see whether a case passes reliably or
only sometimes.  Ids are unique and carry their profile type as a prefix, so --profileType is not needed.`
    },

    parallelTests: {
      type: 'number',
      defaultsTo: 1,
      description: 'Run each case this many times, to see how often it passes rather than whether it passed once.',
      extendedDescription: `Works with any of --naturalLanguageInstructions, --caseId or --all.  The repeats of one case
go out together and the cases stay sequential, so at most this many requests are ever in flight -- launching every case
at once would say more about rate limits than about the prompt.  Results are reported per case as a fail rate, because
a case failing 2 of 5 is a different problem from one failing 5 of 5.`
    },

    validateWithContour: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Check every generated profile with contour, failing a case on any contour error.',
      extendedDescription: `Off by default because it needs the contour binary on PATH, which not every machine has.
Passing it without contour installed is an error rather than a silent skip -- a run that quietly validated nothing
looks exactly like a run where everything was valid.  Apple formats only: contour has no Windows CSP validator, so
csp cases report as not-checked either way.`
    }

  },


  fn: async function ({profileType, naturalLanguageInstructions, all, baseModel, effort, verbose, validateWithContour, parallelTests, caseId}) {

    let path = require('path');
    let util = require('util');

    let waysToChooseWhatToRun = _.compact([all, naturalLanguageInstructions, caseId]).length;
    if(waysToChooseWhatToRun === 0) {
      throw new Error('One of --naturalLanguageInstructions, --caseId or --all is required.  See `sails run test-llm-generated-configuration-profile --help`.');
    }
    if(waysToChooseWhatToRun > 1) {
      throw new Error('--naturalLanguageInstructions, --caseId and --all each choose what to run, so pass exactly one of them.');
    }
    if(parallelTests < 1) {
      throw new Error(`--parallelTests must be at least 1 (got ${parallelTests}).`);
    }

    // Resolved before anything is spent.  Listing near-misses matters with 33 cases: a bare "not found"
    // on a mistyped id means paging back through the file to find the right spelling.
    let chosenCase;
    if(caseId) {
      chosenCase = _.find(TEST_CASES, { id: caseId });
      if(!chosenCase) {
        let nearMisses = _.filter(_.pluck(TEST_CASES, 'id'), (id)=>{
          return _.any(caseId.split('-'), (word)=>{ return word.length > 2 && _.contains(id, word); });
        });
        throw new Error(
          `No case with the id "${caseId}".` +
          (nearMisses.length > 0 ? `\nDid you mean one of:\n  ${nearMisses.join('\n  ')}` : `\nRun with --all to see every case, or check the ids at the top of this script.`)
        );
      }
      if(profileType && profileType !== chosenCase.profileType) {
        throw new Error(`--profileType is "${profileType}" but the case "${caseId}" is a ${chosenCase.profileType} case.  Leave --profileType off; the id already implies it.`);
      }
      profileType = chosenCase.profileType;
    }

    // Checked once, up front, rather than per case: a missing binary is a setup mistake, and finding out
    // about it after a full run has already spent minutes and real money is finding out too late.
    if(validateWithContour) {
      let contourVersion = await sails.helpers.process.executeCommand.with({
        command: 'contour --version',
        timeout: 10000,
      })
      .tolerate(()=>{
        return undefined;
      });
      if(!contourVersion) {
        throw new Error(
          'The --validateWithContour flag was passed, but the `contour` binary could not be run.\n'+
          'Install it and make sure it is on PATH, or re-run without the flag.'
        );
      }
      sails.log(`Validating generated profiles with ${_.trim(contourVersion.stdout) || 'contour'}.`);
    }

    let effortLevelsToRun = [effort];

    // Collected here and written to test-results/baseline/ at the end, in the same shape and under the same filename
    // convention the current script uses, so a baseline transcript and a current one can be read side by side.  The
    // file gets the full per-case output even when the terminal only got a summary.
    let transcriptLines = [];
    let report = (line)=>{ transcriptLines.push(line); sails.log(line); };
    // No push to transcriptLines: the sails.log.warn override installed below already does it, and doing both
    // wrote every warning to the saved file twice.
    let reportWarning = (line)=>{ sails.log.warn(line); };
    // console.log rather than sails.log, for the tables: the log prefix would break column alignment.
    let reportWithoutLogPrefix = (line)=>{ transcriptLines.push(line); console.log(line); };
    let reportToTranscriptOnly = (line)=>{ transcriptLines.push(line); };

    // The prompt helper reports anything it had to work around through sails.log.warn.  Route those into the
    // transcript as well -- on a baseline run they are the record of what the old prompt made the helper do.
    // Restored before the file is written, below.
    let originalSailsLogWarn = sails.log.warn;
    sails.log.warn = function(){
      let args = Array.prototype.slice.call(arguments);
      transcriptLines.push(_.map(args, (arg)=>{ return _.isString(arg) ? arg : util.inspect(arg, {depth: 3}); }).join(' '));
      return originalSailsLogWarn.apply(sails.log, args);
    };

    // A transcript that does not say what produced it cannot be compared against another one, and comparing is the
    // only reason this script saves anything.  "webFetch" is spelled out rather than omitted so the difference from a
    // current-script transcript is stated rather than inferred from an absent line.
    report(
      'Inputs:\n' +
      `profileType: ${profileType || '(every type)'}\n` +
      (all ? 'all: true\n' : caseId ? `caseId: ${caseId}\n` : `naturalLanguageInstructions: ${naturalLanguageInstructions}\n`) +
      `baseModel: ${baseModel}\n` +
      `effort: ${effort || '(model default)'}\n` +
      (parallelTests > 1 ? `parallelTests: ${parallelTests}\n` : '') +
      `verbose: ${verbose}\n` +
      '----------'
    );

    let cases;
    if(all) {
      cases = _.filter(TEST_CASES, (testCase)=>{
        return !profileType || testCase.profileType === profileType;
      });
      if(cases.length === 0) {
        throw new Error(`No cases defined for profileType "${profileType}".`);
      }
    } else {
      // A named case from the list above -- which brings its own expect block, so repeats are scored --
      // or a one-off from --naturalLanguageInstructions, which is not.
      cases = [chosenCase || { id: 'ad-hoc', profileType, instructions: naturalLanguageInstructions }];
    }

    let results = [];

    // Wrapped in an IIFE so the request goes out now rather than when it is awaited: a Sails deferred
    // does not start until something awaits it.  elapsedMs is measured per call, so it stays the model's
    // latency rather than the batch's wall clock.
    let startGenerating = (testCase, effortLevel)=>{
      return (async ()=>{
        let startedAt = Date.now();
        try {
          // The helper supplies the PayloadUUIDs, and it is called once per run, so parallel repeats
          // of one case still get their own set.
          let generatorConfiguration = await sails.helpers.getConfigurationProfileGeneratorConfiguration.with({
            profileType: testCase.profileType,
            naturalLanguageInstructions: testCase.instructions,
          });
          let rawResult = await sails.helpers.ai.prompt.with({
            systemPrompt: generatorConfiguration.systemPrompt,
            prompt: generatorConfiguration.userPrompt,
            baseModel,
            effort: effortLevel,
            expectJson: true,
          });
          return { rawResult, elapsedMs: Date.now() - startedAt };
        } catch (err) {
          return { unexpectedError: err, elapsedMs: Date.now() - startedAt };
        }
      })();
    };

    for (let effortLevel of effortLevelsToRun) {
      for (let testCase of cases) {

        // The repeats of one case go out together; the cases themselves stay sequential.  Thirty cases
        // times five repeats launched at once would measure the rate limit rather than the prompt, and
        // the per-case line printed as each finishes is what keeps a slow run distinguishable from a
        // hung one.  At most parallelTests requests are ever in flight.
        let repeatsInFlight = _.times(parallelTests, ()=>{ return startGenerating(testCase, effortLevel); });

        for (let repeatIdx = 0; repeatIdx < parallelTests; repeatIdx++) {

          // Awaited in order even when they finished out of order, so the transcript reads predictably.
          let outcome = await repeatsInFlight[repeatIdx];
          let rawResult = outcome.rawResult;
          let unexpectedError = outcome.unexpectedError;
          let elapsedMs = outcome.elapsedMs;
          // Numbered only when there is more than one, so a single run still reads as the plain id.
          let displayId = parallelTests > 1 ? `${testCase.id} #${repeatIdx + 1}` : testCase.id;

          // Mirror the action's own acceptance test: an abstention, or a response missing any
          // required key, is not a usable profile.
          let abstained = !unexpectedError && (
            rawResult.couldNotGenerateProfile ||
          !rawResult.configurationProfile ||
          !rawResult.profileFilename ||
          !rawResult.settingsEnforced
          );
          let generatedProfile = (unexpectedError || abstained) ? undefined : {
            profile: rawResult.configurationProfile,
            profileFilename: rawResult.profileFilename,
            deliveryNotes: rawResult.deliveryNotes,
            items: rawResult.settingsEnforced,
          };

          let expectations = testCase.expect || {};
          let checkFailures;
          if(unexpectedError) {
          // Distinguished from an abstention on purpose: collapsing them would hide a
          // misconfigured anthropicSecret as a model refusal.
            checkFailures = [`unexpected error: ${unexpectedError.message}`];
          } else if(abstained) {
            checkFailures = expectations.expectFailure ? [] : [
              `abstained but a profile was expected -- reason given: ${JSON.stringify(rawResult.reasonWhyAProfileCouldNotBeGenerated || '(none)')}`
            ];
          } else {
            checkFailures = checkExpectations(expectations, generatedProfile);
          }

          // Deliberately after elapsedMs is taken: this shells out to another process, and elapsedMs exists to
          // measure the model, not the test harness.
          //
          // Errors fail the case, warnings only get reported.  contour warns about things that are true of a
          // perfectly good profile -- a payload type Apple has since superseded, for instance -- so failing on
          // warnings would turn the suite red for reasons unrelated to what each case is testing.
          let contourResult;
          if(validateWithContour && generatedProfile) {
            contourResult = await runContourValidation(testCase.profileType, generatedProfile.profile);
            checkFailures = checkFailures.concat(_.map(contourResult.errors, (contourError)=>{
              return `contour: ${contourError}`;
            }));
          }

          results.push({
            id: displayId,
            // The base id, so repeats of one case can be grouped back together for the failure rate.
            caseId: testCase.id,
            profileType: testCase.profileType,
            canary: !!testCase.canary,
            effort: effortLevel || '(default)',
            elapsedMs,
            checkFailures,
            generatedProfile,
            contourResult,
          });

          // Report as each case finishes.  A full run takes minutes, and a summary-only script
          // makes a slow run indistinguishable from a hung one.
          report(
          `${String(elapsedMs).padStart(6)}ms  ` +
          `${String(effortLevel || 'default').padEnd(8)} ` +
          `${testCase.profileType.padEnd(13)} ` +
          `${String(displayId).padEnd(38)} ` +
          `${checkFailures.length === 0 ? 'ok' : 'FAILED'}`
          );
          for (let checkFailure of checkFailures) {
            report(`         └─ ${checkFailure}`);
          }
          // A case with only contour warnings still passes, so nothing else would print it.  One line keeps it
          // visible without dragging the whole profile onto the terminal.
          if(contourResult && contourResult.ran && contourResult.errors.length === 0 && contourResult.warnings.length > 0) {
            report(`         └─ contour: ${contourResult.warnings.length} warning(s), not failing the case`);
          }

          // Print the output for anything that failed, for every canary, and for everything when
          // --verbose.  A failure you can't see is a failure you can't act on -- and a canary that
          // passes its substring checks still needs a human, because the properties that matter most
          // on those cases are the ones substrings can't express.
          if(generatedProfile) {
            let banner = testCase.canary && checkFailures.length === 0 ? 'CANARY, automated checks passed -- confirm by eye' : testCase.canary ? 'CANARY, FAILED' : checkFailures.length > 0 ? 'FAILED' : 'ok';
            let detailLines = [
              `\n──── ${displayId} @ effort ${effortLevel || 'default'} (${baseModel}) -- ${banner} ────`,
              `instructions: ${testCase.instructions}`,
            ];
            if(testCase.readByEye) {
              detailLines.push(`CONFIRM BY EYE: ${testCase.readByEye}`);
            }
            detailLines.push(
            `profileFilename: ${generatedProfile.profileFilename}`,
            `deliveryNotes: ${JSON.stringify(generatedProfile.deliveryNotes)}`
            );
            // Stated either way, when it ran at all.  A case contour never looked at and a case contour
            // looked at and liked are very different things, and only one of them is evidence.
            if(!contourResult) {
            // Not validated this run -- say nothing rather than implying a clean bill of health.
            } else if(!contourResult.ran) {
              detailLines.push(`contour: not run -- ${contourResult.skippedBecause}`);
            } else if(contourResult.errors.length === 0 && contourResult.warnings.length === 0) {
              detailLines.push('contour: valid, no findings');
            } else {
              detailLines.push(`contour: ${contourResult.errors.length} error(s), ${contourResult.warnings.length} warning(s)`);
              for (let contourError of contourResult.errors) {
                detailLines.push(`  ERROR   ${contourError}`);
              }
              for (let contourWarning of contourResult.warnings) {
                detailLines.push(`  warning ${contourWarning}`);
              }
            }
            detailLines.push(
            `\n${generatedProfile.profile}\n`,
            `settingsEnforced:\n${util.inspect(generatedProfile.items, {depth: 4, colors: false})}`,
            '────────\n'
            );
            // A focused run -- one case, named or ad-hoc -- prints its detail regardless: it is the only
            // thing there is to look at, and with --parallelTests the variance is read by comparing them.
            let isAdHocRun = !all;
            let writeDetailLine = (verbose || testCase.canary || checkFailures.length > 0 || isAdHocRun) ? report : reportToTranscriptOnly;
            for (let detailLine of detailLines) {
              writeDetailLine(detailLine);
            }
          }
        }
      }
    }

    for (let tableLine of buildResultsTable(results, baseModel, parallelTests)) {
      reportWithoutLogPrefix(tableLine);
    }

    report('\n=== Summary ===');
    for (let effortLevel of effortLevelsToRun) {
      let resultsForThisEffort = _.where(results, { effort: effortLevel || '(default)' });
      let passing = _.filter(resultsForThisEffort, (result)=>{ return result.checkFailures.length === 0; });
      let elapsedTimes = _.pluck(resultsForThisEffort, 'elapsedMs').sort((a, b)=>{ return a - b; });
      report(
        `${baseModel}  effort ${String(effortLevel || 'default').padEnd(8)} ` +
        `passed ${passing.length}/${resultsForThisEffort.length}  ` +
        `median ${elapsedTimes[Math.floor(elapsedTimes.length / 2)]}ms  ` +
        `slowest ${_.last(elapsedTimes)}ms`
      );

      // With repeats the run count buries the thing worth knowing: how many cases are reliable, how
      // many are flaky, and how many never work.  A case failing 2 of 5 is a different problem from
      // one failing 5 of 5, and only the middle group is worth re-running to understand.
      if(parallelTests > 1) {
        let caseIdsInOrder = _.uniq(_.pluck(resultsForThisEffort, 'caseId'));
        let alwaysPassed = [];
        let sometimesFailed = [];
        let alwaysFailed = [];
        for (let caseId of caseIdsInOrder) {
          let runsOfThisCase = _.where(resultsForThisEffort, { caseId });
          let failureCount = _.filter(runsOfThisCase, (result)=>{ return result.checkFailures.length > 0; }).length;
          if(failureCount === 0) { alwaysPassed.push(caseId); }
          else if(failureCount === runsOfThisCase.length) { alwaysFailed.push(caseId); }
          else { sometimesFailed.push({ caseId, failureCount, total: runsOfThisCase.length }); }
        }
        report(
          `${caseIdsInOrder.length} case(s): ${alwaysPassed.length} passed every run, ` +
          `${sometimesFailed.length} flaky, ${alwaysFailed.length} failed every run`
        );
        for (let flaky of sometimesFailed) {
          report(`  flaky   ${flaky.caseId}: failed ${flaky.failureCount}/${flaky.total} (${Math.round(100 * flaky.failureCount / flaky.total)}%)`);
        }
        for (let caseId of alwaysFailed) {
          report(`  always  ${caseId}: failed every run`);
        }
      }
    }

    // Citation resolution can't be checked automatically -- it needs a human against the published
    // reference -- but it degrades before the pass rate does, so surface the raw material.
    let citations = _.uniq(_.flatten(_.map(_.filter(results, 'generatedProfile'), (result)=>{
      return _.map(result.generatedProfile.items || [], (item)=>{ return `${result.id}: ${item.schemaReference}`; });
    })));
    report(`\n${citations.length} citation(s) to spot-check against the published reference:`);
    for (let citation of citations) {
      report(`  ${citation}`);
    }

    // Canaries are the real gate, not the aggregate.  They fail quietly -- a profile that deploys
    // and enforces nothing still looks like a pass to someone skimming output -- so report them
    // separately rather than letting them average out against 30-odd ordinary cases.
    let canaryResults = _.filter(results, 'canary');
    let failedCanaries = _.filter(canaryResults, (result)=>{ return result.checkFailures.length > 0; });
    if(canaryResults.length > 0) {
      if(failedCanaries.length === 0) {
        report(`\nAll ${canaryResults.length} canary run(s) passed their automated checks.  Their full output is printed above -- confirm the "CONFIRM BY EYE" items before calling this a pass, since the properties that matter most on those cases are the ones substring checks cannot express.`);
      } else {
        reportWarning(`\n${failedCanaries.length} of ${canaryResults.length} canary run(s) FAILED.  Treat this as blocking regardless of the aggregate pass rate:`);
        for (let failedCanary of failedCanaries) {
          reportWarning(`  ${failedCanary.id} @ ${failedCanary.effort}: ${failedCanary.checkFailures.join('; ')}`);
        }
      }
    }

    let failedResults = _.filter(results, (result)=>{ return result.checkFailures.length > 0; });
    if(failedResults.length > 0) {
      reportWarning(`\n${failedResults.length} of ${results.length} run(s) failed:`);
      // One line per distinct failure rather than per run: five repeats of one case failing the same
      // way is one problem, not five, and printing it five times hides the cases that failed once.
      let failuresAlreadyReported = {};
      for (let failedResult of failedResults) {
        let reason = failedResult.checkFailures.join('; ');
        let seenBefore = failuresAlreadyReported[failedResult.caseId + reason];
        failuresAlreadyReported[failedResult.caseId + reason] = (seenBefore || 0) + 1;
      }
      let reportedAlready = {};
      for (let failedResult of failedResults) {
        let reason = failedResult.checkFailures.join('; ');
        let key = failedResult.caseId + reason;
        if(reportedAlready[key]) { continue; }
        reportedAlready[key] = true;
        let timesSeen = failuresAlreadyReported[key];
        reportWarning(`  ${failedResult.caseId} @ ${failedResult.effort}${timesSeen > 1 ? ` (x${timesSeen})` : ''}: ${reason}`);
      }
    }

    sails.log.warn = originalSailsLogWarn;

    // Same filename convention as the current script -- format, model, effort, start time -- but under a baseline/
    // subdirectory, so a baseline run and a current run of the same cases sit side by side under the same name.
    // Included so a directory of ad-hoc runs can be told apart without opening them.  Squashed to
    // filename-safe characters and capped well short of the 255-byte limit, since the rest of the name
    // and the timestamp also have to fit.
    let whatWasRunForFilename = '';
    if(caseId) {
      whatWasRunForFilename = ` - ${caseId}`;
    } else if(naturalLanguageInstructions) {
      let squashed = _.trim(naturalLanguageInstructions.replace(/[^a-zA-Z0-9]+/g, '-'), '-');
      if(squashed) {
        whatWasRunForFilename = ` - ${_.trunc(squashed, {length: 60, omission: ''})}`;
      }
    }
    let transcriptPath = path.resolve(
      sails.config.appPath,
      'test-results',
      `${(new Date().toLocaleString()).replace(/\/|\:/g, '-')} - ${profileType || 'all'} - ${baseModel}${whatWasRunForFilename}.txt`
    );
    await sails.helpers.fs.write(transcriptPath, transcriptLines.join('\n'), true);
    sails.log(`\nFull output of this run saved to:\n  ${transcriptPath}`);

  }


};








/**
 * Validate a generated profile with contour, by writing it to a temporary file and shelling out.
 *
 * contour is a macOS toolkit, so what it can check depends on the format: .mobileconfig goes through
 * `contour profile validate`, DDM declarations through `contour profile ddm validate`, and Windows CSP has no
 * contour equivalent at all.  Anything it cannot check comes back `ran: false` rather than clean, so "not
 * checked" never reads as "checked and found nothing".
 *
 * The command has its exit code echoed onto the end because sails.helpers.process.executeCommand treats a
 * non-zero exit as an error and drops stdout on that path -- which is exactly the output worth having, since
 * contour exits non-zero precisely when it found something to report.
 *
 * This catches the defects the substring assertions cannot express: a duplicate PayloadUUID, a value typed as
 * <string>true</string> instead of <true/>, a declaration type that does not exist, a Payload key that is not
 * a real key.  It does not replace the assertions -- contour checks that a profile is well formed and its keys
 * are real, not that it enforces what the admin asked for.
 *
 * @param  {String} profileType
 * @param  {String} profile
 * @returns {Dictionary}  {ran, skippedBecause, errors, warnings}
 */
async function runContourValidation(profileType, profile) {

  let CONTOUR_SUBCOMMAND_BY_PROFILE_TYPE = {
    'mobileconfig': { subcommand: 'profile validate', extension: 'mobileconfig' },
    'ddm': { subcommand: 'profile ddm validate', extension: 'json' },
  };

  let contourSubcommand = CONTOUR_SUBCOMMAND_BY_PROFILE_TYPE[profileType];
  if(!contourSubcommand) {
    return { ran: false, skippedBecause: `contour has no validator for ${profileType} profiles`, errors: [], warnings: [] };
  }

  let os = require('os');
  let path = require('path');
  let util = require('util');

  // Named for the run rather than the case, and removed in the `finally` below, so a crashed run leaves at most
  // one stray file behind rather than one per case.
  let temporaryProfilePath = path.join(
    os.tmpdir(),
    `fleet-generated-profile-${Date.now()}-${Math.random().toString(36).slice(2, 8)}.${contourSubcommand.extension}`
  );

  let rawOutput;
  try {
    await sails.helpers.fs.write(temporaryProfilePath, profile, true);
    let commandResult = await sails.helpers.process.executeCommand.with({
      command: `contour ${contourSubcommand.subcommand} --json '${temporaryProfilePath}'; echo "__CONTOUR_EXIT__$?"`,
      timeout: 30000,
    });
    rawOutput = commandResult.stdout;
  } catch (err) {
    return { ran: false, skippedBecause: `contour could not be run (${err.message})`, errors: [], warnings: [] };
  } finally {
    await sails.helpers.fs.rmrf(temporaryProfilePath).tolerate(()=>{ /* a leftover temp file is not worth failing a run over */ });
  }

  let exitCodeMatch = rawOutput.match(/__CONTOUR_EXIT__(\d+)/);
  let exitCode = exitCodeMatch ? Number(exitCodeMatch[1]) : undefined;
  // Worth its own message: on a machine without contour installed, every single case would otherwise report the
  // same unexplained failure.
  if(exitCode === 127) {
    return { ran: false, skippedBecause: 'contour is not installed on this machine', errors: [], warnings: [] };
  }

  // contour emits the result and then, when it found problems, a second {success:false} object after it -- so
  // the output is a stream of JSON values, not one document.  Only the first carries findings.  Scanned by
  // bracket depth rather than parsed whole, since the trailing object would make JSON.parse fail outright.
  let extractFirstJsonValue = (text)=>{
    let startedAt = text.search(/[[{]/);
    if(startedAt === -1) {
      return undefined;
    }
    let depth = 0;
    let isInsideString = false;
    let isEscaped = false;
    for (let idx = startedAt; idx < text.length; idx++) {
      let character = text[idx];
      if(isInsideString) {
        if(isEscaped) { isEscaped = false; }
        else if(character === '\\') { isEscaped = true; }
        else if(character === '"') { isInsideString = false; }
        continue;
      }
      if(character === '"') { isInsideString = true; }
      else if(character === '{' || character === '[') { depth++; }
      else if(character === '}' || character === ']') {
        depth--;
        if(depth === 0) { return text.slice(startedAt, idx + 1); }
      }
    }
    return undefined;
  };

  let parsedOutput;
  try {
    parsedOutput = JSON.parse(extractFirstJsonValue(rawOutput));
  } catch (unusedErr) {
    return {
      ran: false,
      skippedBecause: `contour exited ${exitCode} and its output could not be parsed: ${rawOutput.slice(0, 200)}`,
      errors: [], warnings: []
    };
  }

  // `profile validate` returns one object, `profile ddm validate` an array of them.
  let contourResults = _.isArray(parsedOutput) ? parsedOutput : [parsedOutput];
  let asText = (finding)=>{
    if(_.isString(finding)) { return finding; }
    if(finding && finding.message) { return finding.field ? `${finding.field}: ${finding.message}` : finding.message; }
    return util.inspect(finding, {depth: 2});
  };

  let errors = [];
  let warnings = [];
  for (let contourResult of contourResults) {
    errors = errors.concat(contourResult.errors || []);
    warnings = warnings.concat(contourResult.warnings || []);
    // Read separately, and deliberately: on a .mobileconfig the top-level "valid" can be true while these
    // schema checks failed, so trusting that flag would let a type mismatch through.
    if(contourResult.schema_validation) {
      errors = errors.concat(contourResult.schema_validation.errors || []);
      warnings = warnings.concat(contourResult.schema_validation.warnings || []);
    }
    // `lint_findings` is deliberately not read: it restates the same findings as `warnings` in a different
    // shape, and reporting both would double every line.
  }

  return { ran: true, errors: _.map(errors, asText), warnings: _.map(warnings, asText) };
}


/**
 * Print the run as a table.
 *
 * Returns lines rather than printing them, because they go to two places: the terminal, without the
 * sails.log prefix that would break column alignment, and the saved transcript.
 *
 * @param  {Array} results
 * @param  {String} baseModel
 * @returns {Array} lines
 */
function buildResultsTable(results, baseModel, parallelTests) {
  // An asterisk rather than a separate column: it keeps the case name and its weight together, and
  // the name column is already the widest thing in the table.
  let nameFor = (name, isCanary)=>{ return isCanary ? `${name} *` : name; };
  let medianOf = (numbers)=>{
    let sorted = numbers.slice().sort((a, b)=>{ return a - b; });
    return sorted[Math.floor(sorted.length / 2)];
  };

  let lines;
  if(parallelTests > 1) {
    // One row per case rather than per run.  With repeats the interesting number is how often a case
    // fails, not what any single run did -- a case that fails two times in five is a different problem
    // from one that fails every time, and a flat list of runs hides which is which.
    let caseIdsInOrder = _.uniq(_.pluck(results, 'caseId'));
    lines = [`\n=== Results (${baseModel}, ${parallelTests} runs per case) ===`].concat(buildTable(
      ['case', 'type', 'passed', 'fail rate', 'median ms'],
      _.map(caseIdsInOrder, (caseId)=>{
        let runsOfThisCase = _.where(results, { caseId });
        let passing = _.filter(runsOfThisCase, (result)=>{ return result.checkFailures.length === 0; });
        let failRate = Math.round(100 * (runsOfThisCase.length - passing.length) / runsOfThisCase.length);
        return [
          nameFor(caseId, _.first(runsOfThisCase).canary),
          _.first(runsOfThisCase).profileType,
          `${passing.length}/${runsOfThisCase.length}`,
          `${failRate}%`,
          String(medianOf(_.pluck(runsOfThisCase, 'elapsedMs'))),
        ];
      }),
      ['left', 'left', 'right', 'right', 'right']
    ));
  } else {
    lines = [`\n=== Results (${baseModel}) ===`].concat(buildTable(
      ['case', 'type', 'ms', 'result'],
      _.map(results, (result)=>{
        return [nameFor(result.id, result.canary), result.profileType, String(result.elapsedMs), result.checkFailures.length === 0 ? 'ok' : `FAIL ${result.checkFailures.length}`];
      }),
      ['left', 'left', 'right', 'left']
    ));
  }

  if(_.some(results, 'canary')) {
    lines.push('* canary -- these fail quietly, so read their output rather than trusting the pass count.');
  }

  return lines;
}

/**
 * Render a box-drawn table, sizing each column to its widest cell.
 *
 * @param  {Array} headers
 * @param  {Array} rows  array of arrays of strings
 * @param  {Array} alignments  'left' or 'right', one per column
 * @returns {Array} lines
 */
function buildTable(headers, rows, alignments) {
  let columnWidths = _.map(headers, (header, columnIndex)=>{
    return _.reduce(rows, (widestSoFar, row)=>{
      return Math.max(widestSoFar, String(row[columnIndex]).length);
    }, header.length);
  });

  let padCell = (value, columnIndex)=>{
    let cell = String(value);
    return alignments[columnIndex] === 'right' ? cell.padStart(columnWidths[columnIndex]) : cell.padEnd(columnWidths[columnIndex]);
  };
  let horizontalRule = (left, join, right)=>{
    return left + _.map(columnWidths, (width)=>{ return '─'.repeat(width + 2); }).join(join) + right;
  };
  let renderRow = (cells)=>{
    return '│ ' + _.map(cells, (cell, columnIndex)=>{ return padCell(cell, columnIndex); }).join(' │ ') + ' │';
  };

  return [
    horizontalRule('┌', '┬', '┐'),
    renderRow(headers),
    horizontalRule('├', '┼', '┤'),
  ].concat(_.map(rows, renderRow), [
    horizontalRule('└', '┴', '┘'),
  ]);
}


/**
 * Compare a generated profile against a case's `expect` block.
 *
 * @param  {Dictionary} expectations
 * @param  {Dictionary} generatedProfile
 * @returns {Array} human-readable descriptions of what failed
 */
function checkExpectations(expectations, generatedProfile) {
  let failures = [];

  if(expectations.expectFailure) {
    return ['expected this request to be refused, but a profile was generated'];
  }

  // Whitespace is stripped from both sides so an assertion doesn't fail on JSON indentation.
  let stripWhitespace = (str)=>{ return String(str).replace(/\s+/g, ''); };
  let profileWithoutWhitespace = stripWhitespace(generatedProfile.profile);

  // Element matchers tolerate attributes.  A plain substring cannot be used for XML elements the
  // prompt requires attributes on: '<Format>int</Format>' never matches the real
  // '<Format xmlns="syncml:metinf">int</Format>'.  That cuts both ways -- as a mustNotContain it
  // would silently miss a wrong format rather than merely reporting a false failure.
  let escapeForRegExp = (str)=>{ return String(str).replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); };
  let elementRegExp = (tagName, textContent)=>{
    return new RegExp('<' + escapeForRegExp(tagName) + '(?:\\s[^>]*)?>\\s*' + escapeForRegExp(textContent) + '\\s*</' + escapeForRegExp(tagName) + '>');
  };

  for (let [tagName, textContent] of expectations.mustContainElement || []) {
    if(!elementRegExp(tagName, textContent).test(generatedProfile.profile)) {
      // Name what was actually there.  Reporting only what was wanted makes a wrong value and an
      // unmatched pattern read identically, and the obvious conclusion is that the matcher is broken
      // rather than the profile -- <Format>chr</Format> where int was expected looks like a regex that
      // cannot cope with the xmlns attribute, right up until you see the value.
      let elementsThatWereThere = generatedProfile.profile.match(
        new RegExp('<' + escapeForRegExp(tagName) + '(?:\\s[^>]*)?>[\\s\\S]*?</' + escapeForRegExp(tagName) + '>', 'g')
      ) || [];
      failures.push(
        `missing expected element: <${tagName}>${textContent}</${tagName}>` +
        (elementsThatWereThere.length > 0
          ? ` -- found ${_.uniq(elementsThatWereThere).slice(0, 3).map((element)=>{ return element.replace(/\s+/g, ' '); }).join(', ')}`
          : ` -- no <${tagName}> element in the profile at all`)
      );
    }
  }

  for (let [tagName, textContent] of expectations.mustNotContainElement || []) {
    if(elementRegExp(tagName, textContent).test(generatedProfile.profile)) {
      failures.push(`contains forbidden element: <${tagName}>${textContent}</${tagName}>`);
    }
  }

  for (let needle of expectations.mustContain || []) {
    if(!_.contains(profileWithoutWhitespace, stripWhitespace(needle))) {
      failures.push(`missing expected substring: ${JSON.stringify(needle)}`);
    }
  }

  for (let needle of expectations.mustNotContain || []) {
    if(_.contains(profileWithoutWhitespace, stripWhitespace(needle))) {
      failures.push(`contains forbidden substring: ${JSON.stringify(needle)}`);
    }
  }

  if(expectations.deliveryNotes === '' && generatedProfile.deliveryNotes !== '') {
    failures.push(`expected empty deliveryNotes, got: ${JSON.stringify(generatedProfile.deliveryNotes)}`);
  }
  if(expectations.deliveryNotesIsNotEmpty && !generatedProfile.deliveryNotes) {
    failures.push('expected a deliveryNotes entry, got an empty string');
  }

  return failures;
}
