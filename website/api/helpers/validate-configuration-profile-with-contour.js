module.exports = {


  friendlyName: 'Validate configuration profile with contour',


  description: 'Validate a generated profile with contour, by writing it to a temporary file and shelling out.',


  extendedDescription:
`contour is a macOS toolkit, so what it can check depends on the format: .mobileconfig goes through
\`contour profile validate\`, DDM declarations through \`contour profile ddm validate\`, and Windows CSP has no
contour equivalent at all.  Anything it cannot check comes back \`ran: false\` rather than clean, so "not
checked" never reads as "checked and found nothing".

This catches the defects the substring assertions cannot express: a duplicate PayloadUUID, a value typed as
<string>true</string> instead of <true/>, a declaration type that does not exist, a Payload key that is not
a real key.  It does not replace the assertions -- contour checks that a profile is well formed and its keys
are real, not that it enforces what the admin asked for.`,


  inputs: {

    profileType: {
      type: 'string',
      required: true,
      isIn: ['mobileconfig', 'ddm', 'csp'],
    },

    profile: {
      type: 'string',
      required: true,
      description: 'The generated profile, exactly as an admin would download it.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Contour result',
      outputDescription: '{ran, skippedBecause, errors: [string], warnings: [string]}',
      outputType: 'ref',
    },

  },


  fn: async function ({profileType, profile}) {

    let os = require('os');
    let path = require('path');
    let util = require('util');

    let CONTOUR_SUBCOMMAND_BY_PROFILE_TYPE = {
      'mobileconfig': { subcommand: 'profile validate', extension: 'mobileconfig' },
      'ddm': { subcommand: 'profile ddm validate', extension: 'json' },
    };

    let contourSubcommand = CONTOUR_SUBCOMMAND_BY_PROFILE_TYPE[profileType];
    if(!contourSubcommand) {
      return { ran: false, skippedBecause: `contour has no validator for ${profileType} profiles`, errors: [], warnings: [] };
    }

    // Named for the run rather than the case, and removed in the `finally` below, so a crashed run leaves at most
    // one stray file behind rather than one per case.
    let temporaryProfilePath = path.join(
      os.tmpdir(),
      `fleet-generated-profile-${Date.now()}-${Math.random().toString(36).slice(2, 8)}.${contourSubcommand.extension}`
    );

    // The exit code is echoed onto the end because sails.helpers.process.executeCommand treats a non-zero exit as
    // an error and drops stdout on that path -- which is exactly the output worth having, since contour exits
    // non-zero precisely when it found something to report.
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


};
