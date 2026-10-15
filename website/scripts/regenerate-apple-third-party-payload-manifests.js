module.exports = {


  friendlyName: 'Regenerate Apple third-party payload manifests',


  description: 'Download every manifest in ProfileManifests\' ManagedPreferencesApplications folder, and write the preference keys they define to website/profile-generator/schema/apple-third-party-payload-manifests.json.',


  extendedDescription:
`Like regenerate-apple-third-party-payload-manifests, but instead of a hand-picked list of apps, this covers
every manifest in https://github.com/ProfileManifests/ProfileManifests/tree/master/Manifests/ManagedPreferencesApplications,
and generates each app's display name and keywords from the manifest itself.

The folder is listed with the GitHub API, so this needs a GitHub access token, set as the githubAccessToken custom config:

  sails_custom__githubAccessToken=YOUR_TOKEN_HERE sails run regenerate-apple-third-party-payload-manifests

Read the diff before committing: ProfileManifests is maintained by hand, and generated keywords are a best guess
at the words an admin would use for an app.`,


  inputs: {

    dry: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Download and parse, but do not write anything.'
    },

  },


  fn: async function ({dry}) {

    let path = require('path');

    if(!sails.config.custom.githubAccessToken) {
      throw new Error('No GitHub access token configured.  Set one with sails_custom__githubAccessToken=YOUR_TOKEN_HERE and try running this script again.');
    }

    let SCHEMA_DIR = path.resolve(sails.config.appPath, 'profile-generator/schema');
    let REPO = 'ProfileManifests/ProfileManifests';
    let BRANCH = 'master';
    let FOLDER_PATH = 'Manifests/ManagedPreferencesApplications';
    let RAW_BASE_URL = `https://raw.githubusercontent.com/${REPO}/${BRANCH}`;

    let baseHeaders = {
      'User-Agent': 'sails run regenerate-apple-third-party-payload-manifests',
      'Authorization': `token ${sails.config.custom.githubAccessToken}`,
    };

    // Past this depth, nesting describes the shape of the manifest rather than anything a profile has to get
    // right.  It matches the depth regenerate-apple-profile-schemas keeps for Apple's own payloads.
    const MAX_DEPTH = 3;

    // Words that say nothing about which app a manifest is for, so are never used as keywords on their own.
    const NON_KEYWORDS = [
      'a', 'an', 'and', 'by', 'for', 'of', 'the', 'to', 'with', 'in', 'on',
      'app', 'apps', 'application', 'applications', 'client', 'agent', 'helper', 'service', 'services', 'desktop', 'tool', 'tools',
      'settings', 'setting', 'preferences', 'preference', 'configuration', 'config', 'configs', 'managed', 'management', 'policy', 'policies',
      'profile', 'payload', 'options', 'general', 'defaults', 'software', 'update', 'updates', 'updater', 'mac', 'macos', 'osx', 'os', 'x', 'enterprise', 'edition', 'plist',
      'browser', 'macapp', 'managedprefs', 'mobile', 'site', 'widget', 'login', 'shares', 'extension', 'installparams', 'pro', 'github',
      'com', 'net', 'org', 'io', 'us', 'de', 'co', 'inc', 'llc', 'ltd',
    ];
    const MAX_KEYWORDS = 3;

    // Product names that are also ordinary words, so on their own they match requests that are not about the
    // app: "edge of the screen" is not Microsoft Edge, "limit ad tracking" is not NoMAD Login AD, and "admin
    // privileges" is not SAP Privileges.  They still count inside a longer keyword, e.g., "connect sync", and
    // KEYWORD_ALIASES gives each app a name that does not collide.
    const GENERIC_KEYWORDS = [
      'ad', 'baseline', 'connect', 'cursor', 'drive', 'edge', 'elevate', 'front', 'google', 'logic', 'motion', 'okta',
      'patch', 'privileges', 'receiver', 'renew', 'super', 'support', 'sync', 'trust', 'verify', 'windows', 'word',
    ];

    // Keywords added to the generated ones, for names an admin uses that the manifest's title and domain do not
    // give, e.g., "Microsoft Defender" for com.microsoft.wdav, whose title is "Microsoft Defender for Endpoint".
    // "microsoft 365 apps" rather than "microsoft 365", which requests about a Microsoft 365 tenant also say.
    // Keyed by preference domain.
    const KEYWORD_ALIASES = {
      'com.alectrona.patch': ['alectrona patch'],
      'com.alectrona.patch-agent': ['alectrona patch agent'],
      'com.apple.logic10': ['logic pro'],
      'com.apple.motionapp': ['apple motion'],
      'com.citrix.receiver.nomas': ['citrix receiver', 'citrix'],
      'com.cloudflare.warp': ['cloudflare'],
      'com.crowdstrike.falcon': ['crowdstrike'],
      'com.frontapp.Front': ['front app'],
      'com.google.drivefs.settings': ['google drive', 'drivefs'],
      'com.google.Keystone': ['google software update'],
      'com.jamf.connect.login': ['jamf connect', 'jamf connect login'],
      'com.jamf.connect.shares': ['jamf connect shares'],
      'com.jamf.connect.sync': ['jamf connect sync'],
      'com.jamf.connect.verify': ['jamf connect verify'],
      'com.jamf.trust': ['jamf trust'],
      'com.jigsaw24.Elevate24': ['jigsaw24 elevate'],
      'com.macjutsu.super': ['superman', 'macjutsu'],
      'com.microsoft.autoupdate2': ['microsoft autoupdate', 'mau', 'microsoft 365 apps', 'office updates'],
      'com.microsoft.Edge': ['microsoft edge'],
      'com.microsoft.office': ['microsoft office', 'microsoft 365 apps', 'office 365'],
      'com.microsoft.rdc.macos': ['windows app', 'remote desktop'],
      'com.microsoft.VSCode': ['vs code'],
      'com.microsoft.wdav': ['defender'],
      'com.microsoft.Word': ['microsoft word'],
      'com.okta.mobile': ['okta verify'],
      'com.okta.mobile.auth-service-extension': ['okta verify'],
      'com.secondsonconsulting.baseline': ['second son baseline'],
      'com.secondsonconsulting.renew': ['second son renew'],
      'com.todesktop.230313mzl4w4u92': ['cursor ide', 'cursor editor'],
      'corp.sap.privileges': ['sap privileges'],
      'menu.nomad.login.ad': ['nomad login'],
      'nl.root3.support': ['supportapp', 'root3'],
      'org.mozilla.firefox': ['mozilla'],
    };

    // List every manifest in the folder.  The trees API lists a whole branch in one request, where the contents API would need one per subfolder.
    let treeUrl = `https://api.github.com/repos/${REPO}/git/trees/${BRANCH}?recursive=1`;
    sails.log(`Listing ${FOLDER_PATH} in ${REPO}...`);
    let treeResponse = await fetch(treeUrl, {headers: baseHeaders});// eslint-disable-line no-undef
    if(!treeResponse.ok) {
      throw new Error(`Listing ${treeUrl} failed with status ${treeResponse.status}.  Check that the GitHub access token is valid and try running this script again.`);
    }
    let tree = await treeResponse.json();
    if(tree.truncated) {
      throw new Error(`GitHub truncated the listing of ${REPO}, so some manifests in ${FOLDER_PATH} would be missed.`);
    }
    let manifestPaths = _.sortBy(_.pluck(_.filter(tree.tree, (item)=>{
      return item.type === 'blob' && _.startsWith(item.path, `${FOLDER_PATH}/`) && _.endsWith(item.path, '.plist');
    }), 'path'), (manifestPath)=>{ return manifestPath.toLowerCase(); });
    if(manifestPaths.length === 0) {
      throw new Error(`${REPO} has no manifests in ${FOLDER_PATH}.  ProfileManifests has probably moved them.`);
    }
    sails.log(`Found ${manifestPaths.length} manifests.`);

    // Parse an XML property list.  Containers (<dict>, <array>) still being filled are kept on a stack, and each
    // value is added to the container on top of it: under the <key> before it in a <dict>, or at the end of an <array>.
    let parsePlist = (plistText, sourceUrl)=>{
      let parsed;
      let openContainers = [];
      // Match each opening, closing or empty plist element, e.g., <dict>, </array>, <true/>.
      let tagRegExp = /<(\/?)(dict|array|key|string|integer|real|date|data|true|false)\b[^>]*?(\/?)>/g;
      let match;
      while ((match = tagRegExp.exec(plistText)) !== null) {
        let isClosingTag = match[1] === '/';
        let tagName = match[2];
        let isEmptyElement = match[3] === '/';
        let value;
        if(isClosingTag) {
          // The closing tags of text elements are skipped over when their text is read below, so only a
          // container's closing tag gets here.
          value = openContainers.pop().value;
        } else if((tagName === 'dict' || tagName === 'array') && !isEmptyElement) {
          openContainers.push({value: tagName === 'dict' ? {} : []});
          continue;
        } else if(tagName === 'dict' || tagName === 'array') {
          value = tagName === 'dict' ? {} : [];
        } else if(tagName === 'true' || tagName === 'false') {
          value = tagName === 'true';
        } else {
          // Read a text element's contents up to its closing tag, and carry on matching after it.
          let text = '';
          if(!isEmptyElement) {
            let closingTagIndex = plistText.indexOf(`</${tagName}>`, tagRegExp.lastIndex);
            if(closingTagIndex === -1) {
              throw new Error(`${sourceUrl} has a <${tagName}> with no closing tag, so it could not be parsed.`);
            }
            text = plistText.slice(tagRegExp.lastIndex, closingTagIndex)
              .replace(/&#(\d+);/g, (unusedMatch, code)=>{ return String.fromCharCode(Number(code)); })
              .replace(/&#x([0-9a-fA-F]+);/g, (unusedMatch, code)=>{ return String.fromCharCode(parseInt(code, 16)); })
              .replace(/&lt;/g, '<')
              .replace(/&gt;/g, '>')
              .replace(/&quot;/g, '"')
              .replace(/&apos;/g, '\'')
              .replace(/&amp;/g, '&');
            tagRegExp.lastIndex = closingTagIndex + tagName.length + 3;
          }
          if(tagName === 'key') {
            _.last(openContainers).pendingKey = text;
            continue;
          }
          value = (tagName === 'integer' || tagName === 'real') ? Number(text) : text;
        }

        let parentContainer = _.last(openContainers);
        if(!parentContainer) {
          parsed = value;
        } else if(_.isArray(parentContainer.value)) {
          parentContainer.value.push(value);
        } else {
          parentContainer.value[parentContainer.pendingKey] = value;
          parentContainer.pendingKey = undefined;
        }
      }
      return parsed;
    };

    // Pick up to MAX_KEYWORDS lowercase words an admin would use for the app in a request, from the manifest's
    // title and preference domain.  e.g., "Google Chrome" (com.google.Chrome) gives ['chrome'], and
    // "Microsoft Defender for Endpoint" (com.microsoft.wdav) gives ['defender for endpoint', 'wdav'].
    let generateKeywords = (manifest)=>{
      let isKeyword = (word)=>{
        return word.length >= 2 && !/^v?\d+$/.test(word) && !_.contains(NON_KEYWORDS, word);
      };
      let domainSegments = _.map(String(manifest.pfm_domain).split('.'), (segment)=>{ return segment.toLowerCase(); });
      // The vendor is the segment after the reverse-DNS prefix, e.g., google in com.google.Chrome.  It is left out of
      // keywords where something more specific is found, so a request that mentions Microsoft is not offered every Microsoft app.
      // For an open-source project's com.github.<owner>.<project> domain, the vendor is the project's owner.
      let vendorIndex = domainSegments[1] === 'github' && domainSegments.length > 3 ? 2 : 1;
      let vendor = domainSegments.length > vendorIndex + 1 ? domainSegments[vendorIndex] : undefined;
      let titleWords = _.compact(String(manifest.pfm_title || '').toLowerCase().replace(/\([^)]*\)/g, ' ').replace(/[^a-z0-9]+/g, ' ').split(' '));
      // Generic words at the end of a title, e.g., settings in "Zoom Settings", are not part of the product's name.
      while (titleWords.length > 0 && !isKeyword(_.last(titleWords))) {
        titleWords.pop();
      }
      let productWords = titleWords[0] === vendor && titleWords.length > 1 ? titleWords.slice(1) : titleWords;
      // A product named only in generic words, e.g., "Software Update" in "Google Software Update", keeps its vendor.
      if(!_.any(productWords, isKeyword)) {
        productWords = titleWords;
      }

      let candidates = [];
      // Keywords are matched as whole words in a request, so a multi-word name stays a phrase, generic words and all, e.g., "defender for endpoint".
      if(productWords.length > 0 && productWords.length <= 3 && _.any(productWords, isKeyword)) {
        candidates.push(productWords.join(' '));
      }
      let productDomainSegments = domainSegments.slice(vendorIndex + 1);
      // The product's first word, e.g., particulars for "Particulars Today Extension" (net.glencode.Particulars.Widget), since admins often
      // shorten a product to it.  Only when the domain uses it too, since a title's first word is often generic, e.g., visual in "Visual Studio Code".
      if(productWords.length > 1 && isKeyword(productWords[0]) && _.contains(productDomainSegments, productWords[0])) {
        candidates.push(productWords[0]);
      }
      // The domain's segments after the vendor, e.g., wdav in com.microsoft.wdav, with any version number dropped, e.g., autoupdate in com.microsoft.autoupdate2.
      // Skipped when they only respell a keyword already picked, e.g., setupmanager for "setup manager" or imovieapp for imovie, or are
      // generated ids, e.g., 230313mzl4w4u92 in com.todesktop.230313mzl4w4u92.
      for (let segment of productDomainSegments) {
        let word = segment.replace(/\d+$/, '');
        let respellsAKeyword = _.any(candidates, (candidate)=>{
          return word.replace(/[^a-z0-9]/g, '') === candidate.replace(/[^a-z0-9]/g, '') || (!_.contains(candidate, ' ') && _.startsWith(word, candidate));
        });
        if(isKeyword(word) && !respellsAKeyword && !/\d{4,}/.test(word)) {
          candidates.push(word);
        }
      }
      if(candidates.length === 0 && vendor && isKeyword(vendor)) {
        candidates.push(vendor);
      }
      candidates = _.reject(candidates, (candidate)=>{ return _.contains(GENERIC_KEYWORDS, candidate); });
      return _.uniq(_.take(candidates, MAX_KEYWORDS).concat(KEYWORD_ALIASES[manifest.pfm_domain] || []));
    };

    let downloadManifest = async (manifestPath)=>{
      let sourceUrl = `${RAW_BASE_URL}/${_.map(manifestPath.split('/'), encodeURIComponent).join('/')}`;
      let response = await fetch(sourceUrl, {headers: baseHeaders});// eslint-disable-line no-undef
      if(!response.ok) {
        throw new Error(`Downloading ${sourceUrl} failed with status ${response.status}.`);
      }
      return {sourceUrl, plistText: await response.text()};
    };

    // Download a few at a time, to finish quickly without tripping GitHub's secondary rate limits.
    let downloads = [];
    for (let batch of _.chunk(manifestPaths, 10)) {
      sails.log(`Downloading ${downloads.length + 1}-${downloads.length + batch.length} of ${manifestPaths.length}...`);
      downloads = downloads.concat(await Promise.all(_.map(batch, downloadManifest)));
    }

    let entries = [];
    let skipped = [];
    for (let {sourceUrl, plistText} of downloads) {

      let manifest;
      try {
        manifest = parsePlist(plistText, sourceUrl);
      } catch (err) {
        skipped.push(`${sourceUrl} (${err.message})`);
        continue;
      }
      if(!manifest || !_.isArray(manifest.pfm_subkeys) || !manifest.pfm_domain) {
        skipped.push(`${sourceUrl} (no pfm_domain or pfm_subkeys)`);
        continue;
      }
      if(_.find(entries, {name: manifest.pfm_domain})) {
        skipped.push(`${sourceUrl} (${manifest.pfm_domain} is already covered by another manifest)`);
        continue;
      }

      // Convert the manifest's keys into keys shaped like apple-payload-manifests.json's.  Each queued item is
      // a list of manifest keys and the array their conversions go in, so a key's subkeys are converted after
      // it, into the subkeys array it was given.
      let keys = [];
      let allConvertedKeys = [];
      let droppedAsRemoved = [];
      let keyListsToConvert = [{pfmKeys: manifest.pfm_subkeys, convertedKeys: keys, depth: 0}];
      while (keyListsToConvert.length > 0) {
        let {pfmKeys, convertedKeys, depth, parentName} = keyListsToConvert.shift();
        // A dictionary whose keys the admin names, e.g., Chrome's ExtensionSettings, which is keyed by extension
        // id, describes its entries with a {{value}} key, and the editor field that names each one with a
        // {{key}} key titled for what the name is, e.g., "Extension ID".
        let entryNamePlaceholder = _.find(pfmKeys, (pfmKey)=>{ return pfmKey.pfm_name === '{{key}}'; });
        for (let pfmKey of pfmKeys) {
          let name = pfmKey.pfm_name;
          // ProfileCreator's own UI controls, e.g., PFC_SegmentedControl_0, which sorts Chrome's keys into tabs,
          // and the {{key}} editor field above.  They are not preferences.  Not every tab control is prefixed
          // PFC_ (Firefox's General_SegmentedControl is not), but every one carries pfm_segments.
          if(_.startsWith(String(name), 'PFC_') || name === '{{key}}' || pfmKey.pfm_segments !== undefined) {
            continue;
          }
          // Every payload's Payload* keys are already in apple-payload-manifests.json, as CommonPayloadKeys.
          if(depth === 0 && _.startsWith(String(name), 'Payload')) {
            continue;
          }
          // Keys hidden from every editor, e.g., Microsoft AutoUpdate's per-app LCID, are written by the app rather than by an admin.
          if(pfmKey.pfm_hidden === 'all' || pfmKey.pfm_hidden === 'hidden') {
            continue;
          }
          // pfm_app_max is the last version of the app that reads the key, so setting it does nothing.
          if(pfmKey.pfm_app_max !== undefined) {
            droppedAsRemoved.push(`${parentName ? `${parentName}/` : ''}${name} (last read by ${pfmKey.pfm_app_max})`);
            continue;
          }
          if(name === undefined) {
            // An array's element.  Named the way Apple names theirs, e.g., AllowListItem.
            name = `${parentName}Item`;
          } else if(name === '{{value}}') {
            name = `<${(entryNamePlaceholder && entryNamePlaceholder.pfm_title) || 'any name'}>`;
          }

          // Keep the leading whole sentences of the key's description that fit in 250 characters, cutting the first one only when it is too long on its own.
          let sentences = String(pfmKey.pfm_description || '').replace(/\s+/g, ' ').trim().split(/(?<=[a-z0-9)]\.)\s+(?=[A-Z])/);
          let description = sentences[0];
          for (let sentence of sentences.slice(1)) {
            if(`${description} ${sentence}`.length > 250) {
              break;
            }
            description = `${description} ${sentence}`;
          }
          if(description.length > 250) {
            description = description.slice(0, 250).replace(/\s+\S*$/, '') + '…';
          }

          let rangeList = _.isArray(pfmKey.pfm_range_list) ? pfmKey.pfm_range_list : undefined;
          let rangeListTitles = _.isArray(pfmKey.pfm_range_list_titles) ? pfmKey.pfm_range_list_titles : undefined;
          let converted = _.omit({
            key: name,
            type: pfmKey.pfm_type,
            required: pfmKey.pfm_require === 'always' || undefined,
            allowedValues: rangeList,
            // What each allowed value means, in the same order, since Chrome and Edge's integer settings say nothing on their own, e.g., DefaultCookiesSetting's 1, 2 and 4.
            allowedValueTitles: (rangeList && rangeListTitles && rangeListTitles.length === rangeList.length && _.any(rangeListTitles, (title, idx)=>{ return title !== String(rangeList[idx]); })) ? rangeListTitles : undefined,
            min: pfmKey.pfm_range_min,
            max: pfmKey.pfm_range_max,
            // Only plain values: a few defaults are whole dictionaries, which document the manifest more than the key.
            default: (_.isString(pfmKey.pfm_default) && pfmKey.pfm_default !== '') || _.isNumber(pfmKey.pfm_default) || _.isBoolean(pfmKey.pfm_default) ? pfmKey.pfm_default : undefined,
            format: pfmKey.pfm_format,
            deprecated: pfmKey.pfm_app_deprecated !== undefined || undefined,
            // The key's title and description, since the title is often what says what a key is about, e.g., "Default clipboard setting".
            description: _.compact([pfmKey.pfm_title, description]).join(' -- ') || undefined,
          }, _.isUndefined);

          if(_.isArray(pfmKey.pfm_subkeys) && pfmKey.pfm_subkeys.length > 0 && depth < MAX_DEPTH) {
            converted.subkeys = [];
            keyListsToConvert.push({pfmKeys: pfmKey.pfm_subkeys, convertedKeys: converted.subkeys, depth: depth + 1, parentName: name});
          }
          convertedKeys.push(converted);
          allConvertedKeys.push(converted);
        }
      }
      // Remove the subkeys of a key whose subkeys were all left out above.
      for (let convertedKey of allConvertedKeys) {
        if(convertedKey.subkeys && convertedKey.subkeys.length === 0) {
          delete convertedKey.subkeys;
        }
      }

      if(droppedAsRemoved.length > 0) {
        sails.log.verbose(`${manifest.pfm_domain}: dropped ${droppedAsRemoved.length} keys the app no longer reads: ${droppedAsRemoved.join(', ')}`);
      }
      if(keys.length === 0) {
        skipped.push(`${sourceUrl} (no keys an admin can set)`);
        continue;
      }

      let keywords = generateKeywords(manifest);
      if(keywords.length === 0) {
        sails.log.warn(`${manifest.pfm_domain}: could not generate any keywords from its title (${manifest.pfm_title}) or domain, so a request will never be offered its keys.`);
      }

      entries.push(_.omit({
        name: manifest.pfm_domain,
        displayName: manifest.pfm_title ? String(manifest.pfm_title).trim() : manifest.pfm_domain,
        keywords,
        title: manifest.pfm_title,
        description: manifest.pfm_description ? String(manifest.pfm_description).replace(/\s+/g, ' ').trim() : undefined,
        platforms: manifest.pfm_platforms,
        // ProfileManifests' targets as the PayloadScope values a profile would use, e.g., system is System.
        payloadScopes: _.isArray(manifest.pfm_targets) ? _.map(manifest.pfm_targets, (target)=>{ return _.capitalize(target); }) : undefined,
        manifestVersion: manifest.pfm_version,
        lastModified: manifest.pfm_last_modified ? String(manifest.pfm_last_modified).slice(0, 10) : undefined,
        sourceUrl,
        documentationUrl: manifest.pfm_documentation_url,
        keys,
      }, _.isUndefined));
      sails.log(`${manifest.pfm_domain} v${manifest.pfm_version} [${keywords.join(', ')}]: ${keys.length} top-level keys, ${allConvertedKeys.length} in all.`);
    }

    if(skipped.length > 0) {
      sails.log.warn(`Skipped ${skipped.length} manifests:\n  ${skipped.join('\n  ')}`);
    }
    if(entries.length === 0) {
      throw new Error('Refusing to write: no manifests produced any keys.  ProfileManifests has probably changed its format.');
    }

    let outputPath = path.join(SCHEMA_DIR, 'apple-third-party-payload-manifests.json');
    if(dry) {
      sails.log(`Dry run -- not writing.  Would have written ${entries.length} payloads to ${outputPath}.`);
      return;
    }

    await sails.helpers.fs.writeJson.with({
      destination: outputPath,
      json: {
        generatedAt: (new Date()).toISOString(),
        source: `https://github.com/${REPO}/tree/${BRANCH}/${FOLDER_PATH}`,
        // Sorted by manifest file name, and each app's keys in its manifest's order, which groups related keys together.
        entries,
      },
      force: true
    });

    sails.log(`\nWrote ${entries.length} payloads to ${outputPath}.  Read the diff before committing.`);

  }


};
