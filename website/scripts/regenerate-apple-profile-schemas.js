module.exports = {


  friendlyName: 'Regenerate Apple profile schemas',


  description: 'Read Apple\'s published payload manifests and declaration definitions, and save the keys the configuration profile generator needs to website/profile-generator/schema/apple-payload-manifests.json and website/profile-generator/schema/apple-ddm-declarations.json.',


  extendedDescription:
`The .mobileconfig and DDM schemas in the profile generator's prompt were maintained by hand, and gave key
names and types only.  That is enough to stop the model inventing a key, and it is not enough to stop it
picking the wrong real one: asked to show a list of users at the login window, it emitted
SHOWOTHERUSERS_MANAGED -- a real key sitting six characters away from SHOWFULLNAME on the same line --
because nothing in the schema said what either key does.

Apple documents what they do, so this reads it from the source rather than leaving it to recall:

  SHOWFULLNAME  If true, the system shows the name and password dialog. If false, the system displays a
                list of users.
  ExcludedPaths  ...relative to the user's home directory... Directory paths need to include a trailing "/".

Both of those are failures the generator's own test cases hit, the second on every model and every run.

Only some keys earn that treatment.  Glossing all 1,840 of them roughly triples the schema, and most of
the additions restate the key name -- allowCamera does not need a sentence explaining that false prevents
the camera.  So the criteria are narrow by design: allowed-value lists and numeric ranges, value formats
documented in prose, and the handful of booleans whose two branches do different things rather than one
being the absence of the other.  See gloss() below.

Run this when Apple publishes new payloads or declarations, and read the diff before committing.`,


  inputs: {

    dry: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Parse and report, but do not write the output files.'
    },

  },


  fn: async function ({dry}) {

    let path = require('path');
    let YAML = require('yaml');

    const GITHUB_API_BASE_URL = 'https://api.github.com/repos/apple/device-management';
    const RAW_BASE_URL = 'https://raw.githubusercontent.com/apple/device-management/release';

    // Both schemas come from one repo, so they are regenerated together -- their upstream moves as a unit,
    // and a run that refreshed one but not the other would be a state nobody asked for.
    let SCHEMAS_TO_BUILD = [
      {
        label: 'mobileconfig payload manifests',
        directory: 'mdm/profiles',
        outputFilename: 'apple-payload-manifests.json',
        nameFrom: (doc, filename)=>{ return (doc.payload && doc.payload.payloadtype) || filename.replace(/\.yaml$/, ''); },
        // Sized off the current published set, with headroom for Apple adding or retiring payloads.
        minimumPlausibleCount: 100,
        mustInclude: ['com.apple.loginwindow/SHOWFULLNAME', 'com.apple.mobiledevice.passwordpolicy/forcePIN', 'com.apple.applicationaccess/allowCamera'],
      },
      {
        label: 'ddm declarations',
        directory: 'declarative/declarations/configurations',
        outputFilename: 'apple-ddm-declarations.json',
        nameFrom: (doc, filename)=>{
          // Declaration YAMLs are named without the com.apple.configuration prefix their type carries.
          return (doc.payload && (doc.payload.declarationtype || doc.payload.payloadtype)) || `com.apple.configuration.${filename.replace(/\.yaml$/, '')}`;
        },
        minimumPlausibleCount: 40,
        mustInclude: ['com.apple.configuration.migration-assistant.settings/ExcludedPaths', 'com.apple.configuration.passcode.settings/MinimumLength'],
      },
    ];

    let results = [];
    for (let schemaToBuild of SCHEMAS_TO_BUILD) {
      sails.log(`\nReading ${schemaToBuild.label} from apple/device-management...`);

      // The git trees API rather than the contents API, because a listing that comes back short has to be
      // detectable.  The contents API ignores per_page and page -- it answers with the whole directory
      // however it is asked -- and stops at 1,000 files with nothing in the response to say so, which
      // would quietly drop payloads from a file this script then overwrites.  A tree is addressed as
      // `<ref>:<path>` and carries `truncated` for the same failure at its own (much larger) ceiling.
      //
      // GitHub rejects API requests with no User-Agent, so that header is required rather than decorative.
      // No token: this reads a public repo, and needing credentials to regenerate a schema would be a
      // reason not to regenerate it.
      let tree = await sails.helpers.http.get(`${GITHUB_API_BASE_URL}/git/trees/release:${schemaToBuild.directory}`, {}, {
        'User-Agent': 'Fleet configuration profile generator schema build',
        'Accept': 'application/vnd.github+json',
      })
      .intercept((err)=>{
        return new Error(
          `Could not list ${schemaToBuild.directory} in apple/device-management.  GitHub rate-limits ` +
          `unauthenticated API requests to 60 an hour, so if this has been run a few times already, wait ` +
          `and try again.  Full error: ${err.message}`
        );
      });

      if(tree.truncated) {
        throw new Error(
          `Refusing to overwrite ${schemaToBuild.outputFilename}: GitHub truncated its listing of ` +
          `${schemaToBuild.directory}, so an unknown number of payloads were never offered to this script.  ` +
          `That directory has outgrown a single tree request and this script needs to walk it in pieces.`
        );
      }

      let filenames = _.filter(_.pluck(tree.tree, 'path'), (name)=>{ return _.endsWith(name, '.yaml'); }).sort();
      sails.log(`  ${filenames.length} file(s) to read.`);

      // Eight at a time against raw.githubusercontent, which is a CDN and does not object to this the way
      // the API would.  Nothing here is worth the complexity the Windows scraper needs -- these are static
      // files rather than rendered pages, and a failure is a plain miss rather than a throttle.
      let entries = [];
      let filesThatFailed = [];
      let CONCURRENCY = 8;
      for (let batch of _.chunk(filenames, CONCURRENCY)) {
        await sails.helpers.flow.simultaneouslyForEach(batch, async (filename)=>{
          let rawYaml;
          for (let attempt = 1; attempt <= 3 && rawYaml === undefined; attempt++) {
            if(attempt > 1) {
              await sails.helpers.flow.pause(1000 * attempt);
            }
            rawYaml = await sails.helpers.http.get(`${RAW_BASE_URL}/${schemaToBuild.directory}/${filename}`).tolerate(()=>{ return undefined; });
          }
          if(rawYaml === undefined) {
            filesThatFailed.push(filename);
            return;
          }
          let doc;
          try {
            doc = YAML.parse(rawYaml);
          } catch (err) {
            filesThatFailed.push(`${filename} (could not be parsed as YAML: ${err.message})`);
            return;
          }
          if(!doc || !_.isArray(doc.payloadkeys)) {
            return;// Not every file in these directories describes a payload -- skip quietly.
          }
          entries.push({
            name: schemaToBuild.nameFrom(doc, filename),
            sourceFile: `${schemaToBuild.directory}/${filename}`,
            keys: extractKeys(doc.payloadkeys, 0, []),
          });
        });
      }

      if(filesThatFailed.length > 0) {
        throw new Error(
          `Refusing to overwrite ${schemaToBuild.outputFilename}: ${filesThatFailed.length} file(s) could not ` +
          `be read:\n  ${filesThatFailed.join('\n  ')}`
        );
      }
      if(entries.length < schemaToBuild.minimumPlausibleCount) {
        throw new Error(
          `Refusing to overwrite ${schemaToBuild.outputFilename}: only ${entries.length} entries parsed, below ` +
          `the ${schemaToBuild.minimumPlausibleCount} floor.  Apple has most likely changed the shape of these ` +
          `files, so the parser needs updating rather than the data.`
        );
      }

      let missing = [];
      for (let expectedPath of schemaToBuild.mustInclude) {
        let [entryName, keyName] = expectedPath.split('/');
        let entry = _.find(entries, { name: entryName });
        if(!entry || !_.find(entry.keys, { key: keyName })) {
          missing.push(expectedPath);
        }
      }
      if(missing.length > 0) {
        throw new Error(
          `Refusing to overwrite ${schemaToBuild.outputFilename}: ${missing.length} key(s) the profile ` +
          `generator's test cases depend on did not parse:\n  ${missing.join('\n  ')}`
        );
      }

      // Sorted by source file as well as by name so that two runs of this script produce the same file.
      // Entries arrive in whatever order the concurrent fetches finish in, and eight of them share a name
      // -- six payloads are com.apple.MCX -- so a sort on name alone leaves those in arrival order and
      // reshuffles them run to run.  That turns a regeneration into a 900-line diff of moved text, which
      // is the one thing this script asks a human to read.
      let sortedEntries = _.sortByAll(entries, ['name', 'sourceFile']);
      sails.log(`  ${sortedEntries.length} entries, ${_.reduce(sortedEntries, (total, entry)=>{ return total + countKeys(entry.keys); }, 0)} keys.`);
      results.push({schemaToBuild, sortedEntries});
    }

    if(dry) {
      sails.log('\nDry run -- not writing.');
      return;
    }

    for (let {schemaToBuild, sortedEntries} of results) {
      let outputPath = path.resolve(sails.config.appPath, `profile-generator/schema/${schemaToBuild.outputFilename}`);
      await sails.helpers.fs.writeJson.with({
        destination: outputPath,
        json: {
          generatedAt: (new Date()).toISOString(),
          source: `https://github.com/apple/device-management/tree/release/${schemaToBuild.directory}`,
          entries: sortedEntries,
        },
        force: true
      });
      sails.log(`Wrote ${sortedEntries.length} entries to ${outputPath}.`);
    }

    sails.log('\nRead the diff before committing -- these files are what the generator tells the model is true.');

  }


};


/**
 * Pull the keys worth keeping out of one payload's `payloadkeys`.
 *
 * Apple's manifests carry far more than a generator needs -- per-OS availability, supervision and
 * enrollment modes, titles, update behaviour -- so this keeps the name, the type, whether it is required,
 * and whatever constrains the value, and drops the rest.
 *
 * Depth is capped and visited keys are tracked because a few manifests are self-referential: YAML anchors
 * resolve to shared objects, and walking them naively never terminates.
 *
 * @param  {Array} payloadKeys
 * @param  {Number} depth
 * @param  {Array} alreadyVisited
 * @returns {Array}
 */
function extractKeys(payloadKeys, depth, alreadyVisited) {
  const MAX_DEPTH = 3;
  if(depth > MAX_DEPTH) {
    return [];
  }
  let keys = [];
  for (let payloadKey of payloadKeys || []) {
    // Keys named with a leading underscore are Apple's placeholders for "an element of the array above"
    // rather than keys an admin writes.
    if(!payloadKey.key || _.startsWith(payloadKey.key, '_') || _.contains(alreadyVisited, payloadKey)) {
      continue;
    }
    alreadyVisited.push(payloadKey);

    let range = payloadKey.range || {};
    let extracted = {
      key: payloadKey.key,
      type: String(payloadKey.type || '').replace(/[<>]/g, '') || undefined,
      required: payloadKey.presence === 'required' || undefined,
      allowedValues: _.isArray(payloadKey.rangelist) ? payloadKey.rangelist : undefined,
      min: range.min,
      max: range.max,
      default: payloadKey.default,
      gloss: gloss(payloadKey),
    };
    let nested = extractKeys(payloadKey.subkeys, depth + 1, alreadyVisited);
    if(nested.length > 0) {
      extracted.subkeys = nested;
    }
    keys.push(_.omit(extracted, _.isUndefined));
  }
  return keys;
}


/**
 * Decide whether a key needs a sentence of explanation, and produce it.
 *
 * Every gloss is a cost paid on every request, so this is deliberately narrow.  Two things qualify:
 *
 *   - A value format documented in prose.  ExcludedPaths is an `<array>` of `<string>`, which says nothing
 *     about entries being home-relative with a trailing slash -- the detail every model gets wrong.
 *   - A boolean whose documentation describes both branches.  Apple writes both out exactly when the false
 *     branch is not simply the absence of the true one, which is where a key name misleads: SHOWFULLNAME
 *     false shows a list of users, and no amount of staring at the name tells you that.
 *
 * Allowed-value lists and ranges are carried as structured fields instead, so they need no prose.
 *
 * @param  {Dictionary} payloadKey
 * @returns {String|undefined}
 */
function gloss(payloadKey) {
  let type = String(payloadKey.type || '');
  let content = String(payloadKey.content || '');
  if(!content) {
    return undefined;
  }

  const DOCUMENTS_A_VALUE_FORMAT = /relative to|trailing|must be|must include|separated by|e\.g\.|For example|in the form/i;
  const DESCRIBES_BOTH_BRANCHES = [/if\s+`?true/i, /if\s+`?false/i];

  if(/string|array/.test(type) && DOCUMENTS_A_VALUE_FORMAT.test(content)) {
    // Kept by sentence rather than by character count: the constraint is often in the second sentence, and
    // ExcludedPaths' trailing-slash rule is exactly the part a character cap cuts off.
    return sentencesMatching(content, DOCUMENTS_A_VALUE_FORMAT, 260);
  }
  if(/boolean/.test(type) && _.every(DESCRIBES_BOTH_BRANCHES, (branch)=>{ return branch.test(content); })) {
    return squash(content, 150);
  }
  return undefined;
}


/**
 * Join the sentences of `content` that match `regExp`, up to `maxLength`.
 *
 * @param  {String} content
 * @param  {RegExp} regExp
 * @param  {Number} maxLength
 * @returns {String}
 */
function sentencesMatching(content, regExp, maxLength) {
  let sentences = squash(content, Number.MAX_SAFE_INTEGER).split(/(?<=\.)\s+/);
  let kept = [];
  for (let sentence of sentences) {
    if(regExp.test(sentence)) {
      kept.push(sentence.trim());
    }
    if(kept.join(' ').length > maxLength) {
      break;
    }
  }
  return squash((kept.length > 0 ? kept : [_.first(sentences)]).join(' '), maxLength);
}


/**
 * Collapse whitespace and backticks, and cut to `maxLength` on a word boundary.
 *
 * @param  {String} str
 * @param  {Number} maxLength
 * @returns {String}
 */
function squash(str, maxLength) {
  let squashed = String(str).replace(/`/g, '').replace(/\s+/g, ' ').trim();
  if(squashed.length <= maxLength) {
    return squashed;
  }
  return squashed.slice(0, maxLength).replace(/\s+\S*$/, '') + '…';
}


/**
 * Count keys in a tree, for the log line.
 *
 * @param  {Array} keys
 * @returns {Number}
 */
function countKeys(keys) {
  return _.reduce(keys, (total, key)=>{ return total + 1 + (key.subkeys ? countKeys(key.subkeys) : 0); }, 0);
}
