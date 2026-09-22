module.exports = {


  friendlyName: 'Regenerate Windows CSP policy nodes',


  description: 'Scrape Microsoft\'s published Policy CSP reference and save every node\'s LocURI, format, access type, and allowed values to website/profile-generator/schema/windows-csp-policy-nodes.json.',


  extendedDescription:
`The configuration profile generator has an embedded schema for .mobileconfig payloads and DDM
declarations, but until now nothing for Windows CSP -- it was given a documentation URL and asked to
recall node paths from memory.  It recalled them wrong: the wrong area segment (System/AllowClipboardHistory
for Experience/AllowClipboardHistory), a truncated prefix (./DeviceLock/DevicePasswordEnabled), and
inverted booleans on nodes where 0 means enabled.

Microsoft publishes the format, access type, default, allowed values and dependencies for every node on
the per-area Policy CSP pages, so this walks all of them and writes the result out as data.  The scrape
is the only way in: the markdown behind those pages lives in MicrosoftDocs/windows-docs-pr, which is private.

Because it parses rendered HTML, it fails loudly rather than quietly: if a page's markup changes shape,
the sanity check at the end refuses to overwrite the committed file.  Re-run it when Windows ships a new
policy surface, and read the diff before committing.`,


  inputs: {

    dry: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Parse and report, but do not write the output file.'
    },

  },


  fn: async function ({dry}) {

    let path = require('path');
    let LEARN_BASE_URL = 'https://learn.microsoft.com/en-us/windows/client-management/mdm';

    // Every node the profile generator's own test cases depend on.  These are the assertions that were
    // failing before this file existed, so if the parser stops finding any of them it has regressed in
    // exactly the way that matters, whatever the total node count says.
    let NODES_THAT_MUST_PARSE = {
      'DeviceLock/DevicePasswordEnabled': {format: 'int'},
      'DeviceLock/MinDevicePasswordLength': {format: 'int'},
      'DeviceLock/MaxDevicePasswordFailedAttempts': {format: 'int'},
      'Storage/RemovableDiskDenyWriteAccess': {format: 'int'},
      'Experience/AllowClipboardHistory': {format: 'int'},
      'System/AllowTelemetry': {format: 'int'},
      'ApplicationManagement/AllowAppStoreAutoUpdate': {format: 'int'},
      'WindowsAI/DisableAIDataAnalysis': {format: 'int'},
      'LocalPoliciesSecurityOptions/InteractiveLogon_MessageTextForUsersAttemptingToLogOn': {format: 'chr'},
    };

    // A scrape that half-works is worse than one that fails, because it writes a file that looks fine and
    // is missing a third of the surface.  Sized off the current published set (269 areas, ~3,100 nodes)
    // with enough headroom that Microsoft adding or retiring policies does not trip it.
    let MINIMUM_PLAUSIBLE_AREA_COUNT = 200;
    let MINIMUM_PLAUSIBLE_NODE_COUNT = 2500;

    let areaSlugs = await getPolicyAreaSlugs(LEARN_BASE_URL);
    sails.log(`Found ${areaSlugs.length} Policy CSP area pages.`);

    // Microsoft's CDN throttles a sustained burst rather than a fast one: at eight concurrent it starts
    // erroring a third of the way into a 269-page run, and even at four the last twenty areas fail.  So
    // the first pass goes wide and a second pass picks up whatever it dropped, one at a time with long
    // waits.  A dropped page is indistinguishable from an area with no nodes once parsed, which is why
    // this has to be reliable here rather than checked downstream.
    let nodes = [];
    let CONCURRENCY = 4;

    let fetchArea = async (areaSlug)=>{
      let pageHtml = await fetchFromLearn(`${LEARN_BASE_URL}/policy-csp-${areaSlug}`);
      if(!pageHtml) {
        return false;
      }
      nodes = nodes.concat(parseAreaPage(pageHtml));
      return true;
    };

    let areasToRetry = [];
    let areasFetchedSoFar = 0;
    for (let batch of _.chunk(areaSlugs, CONCURRENCY)) {
      await sails.helpers.flow.simultaneouslyForEach(batch, async (areaSlug)=>{
        if(!await fetchArea(areaSlug)) {
          areasToRetry.push(areaSlug);
        }
      });
      areasFetchedSoFar += batch.length;
      sails.log(`  ...${areasFetchedSoFar}/${areaSlugs.length} areas`);
    }

    let areasThatFailed = [];
    if(areasToRetry.length > 0) {
      sails.log(`Retrying ${areasToRetry.length} area(s) the first pass could not fetch, one at a time...`);
      for (let areaSlug of areasToRetry) {
        let fetched = false;
        for (let attempt = 1; attempt <= 5 && !fetched; attempt++) {
          await sails.helpers.flow.pause(2000 * attempt);
          fetched = await fetchArea(areaSlug);
        }
        if(!fetched) {
          areasThatFailed.push(areaSlug);
        }
      }
    }

    // Hard failure, not a warning.  An area that never arrived parses as zero nodes, so letting the run
    // continue writes a file that is quietly missing a chunk of the policy surface.
    if(areasThatFailed.length > 0) {
      throw new Error(
        `Refusing to overwrite the committed node file: ${areasThatFailed.length} area page(s) could not be ` +
        `fetched, even after retrying them individually:\n  ${areasThatFailed.join('\n  ')}\n\n` +
        `If those say 429, this is rate limiting rather than anything wrong with the script -- one full run ` +
        `is 270 requests, and Microsoft starts refusing for several minutes after a couple of runs ` +
        `back to back.  Wait, then run it again.`
      );
    }

    // Every real node documents a format.  A section with a path but no format is prose that happens to
    // mention a LocURI -- the UserRights page's "General example" heading, for one -- so the format is
    // what separates a policy from a worked example, not the path.
    let nodesWithAPath = _.filter(nodes, (node)=>{ return node.locUri && node.format; });
    let areaNames = _.uniq(_.pluck(nodesWithAPath, 'area'));
    sails.log(`Parsed ${nodesWithAPath.length} nodes across ${areaNames.length} areas.`);

    let missing = [];
    for (let expectedPath of Object.keys(NODES_THAT_MUST_PARSE)) {
      let found = _.find(nodesWithAPath, (node)=>{ return `${node.area}/${node.name}` === expectedPath; });
      if(!found) {
        missing.push(`${expectedPath} -- not found at all`);
      } else if(found.format !== NODES_THAT_MUST_PARSE[expectedPath].format) {
        missing.push(`${expectedPath} -- parsed format "${found.format}", expected "${NODES_THAT_MUST_PARSE[expectedPath].format}"`);
      }
    }

    if(areaNames.length < MINIMUM_PLAUSIBLE_AREA_COUNT || nodesWithAPath.length < MINIMUM_PLAUSIBLE_NODE_COUNT) {
      throw new Error(
        `Refusing to overwrite the committed node file: this run parsed ${nodesWithAPath.length} nodes across ` +
        `${areaNames.length} areas, below the ${MINIMUM_PLAUSIBLE_NODE_COUNT}/${MINIMUM_PLAUSIBLE_AREA_COUNT} floor.  ` +
        `Microsoft has most likely changed the markup these pages are built from, so the parser needs updating ` +
        `rather than the data.`
      );
    }
    if(missing.length > 0) {
      throw new Error(
        `Refusing to overwrite the committed node file: ${missing.length} node(s) the profile generator's ` +
        `test cases depend on did not parse as expected:\n  ${missing.join('\n  ')}`
      );
    }

    let outputPath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-csp-policy-nodes.json');
    if(dry) {
      sails.log(`Dry run -- not writing.  Would have written ${nodesWithAPath.length} nodes to ${outputPath}.`);
      return;
    }

    // Sorted so a regeneration produces a readable diff instead of reshuffling 3,000 lines whenever
    // Microsoft reorders a page.
    let sortedNodes = _.sortBy(nodesWithAPath, (node)=>{ return `${node.area}/${node.name}`; });
    await sails.helpers.fs.writeJson.with({
      destination: outputPath,
      json: {
        generatedAt: (new Date()).toISOString(),
        source: `${LEARN_BASE_URL}/policy-configuration-service-provider`,
        nodes: sortedNodes,
      },
      force: true
    });

    sails.log(`\nWrote ${sortedNodes.length} Policy CSP nodes to ${outputPath}.`);
    sails.log(`Read the diff before committing -- this file is scraped from rendered HTML, so a markup change shows up as content.`);

  }


};


/**
 * Get the slug of every Policy CSP area page from the MDM table of contents.
 *
 * @param  {String} learnBaseUrl
 * @returns {Array} area slugs, e.g. ['devicelock', 'storage', ...]
 */
async function getPolicyAreaSlugs(learnBaseUrl) {

  // Retried like the area pages are.  This is one request out of 270, but it is the one that decides
  // whether the run happens at all, and losing it to a throttled moment fails the script with a bare
  // "non200Response" rather than anything a reader could act on.
  let tableOfContents = await fetchFromLearn(`${learnBaseUrl}/toc.json`, 5);
  if(!tableOfContents) {
    throw new Error(
      `Could not fetch the MDM table of contents at ${learnBaseUrl}/toc.json, so there is no list of ` +
      `Policy CSP areas to walk.  A 429 here means rate limiting rather than anything wrong with the ` +
      `script: a full run is 270 requests and Microsoft refuses for several minutes after a couple of ` +
      `runs back to back.  Wait, then try again.`
    );
  }

  // The tree is deeply nested and its shape is not contractual, so collect hrefs wherever they appear
  // rather than walking a fixed path to them.
  let hrefs = [];
  let collectHrefs = (branch)=>{
    if(_.isArray(branch)) {
      _.each(branch, collectHrefs);
    } else if(_.isObject(branch)) {
      if(_.isString(branch.href)) {
        hrefs.push(branch.href);
      }
      _.each(_.values(branch), collectHrefs);
    }
  };
  collectHrefs(tableOfContents);

  // Anchored to the start of the href on purpose.  Five pages are named "policies-in-policy-csp-supported-by-…"
  // -- roundups of which policies apply to HoloLens and Surface Hub, not area pages -- and an unanchored
  // match pulls "supported-by-hololens2" out of them and then 404s on it.
  let slugs = [];
  for (let href of hrefs) {
    let slugMatch = href.match(/^policy-csp-([a-z0-9-]+)$/);
    if(slugMatch) {
      slugs.push(slugMatch[1]);
    }
  }
  return _.uniq(slugs);
}


/**
 * GET a learn.microsoft.com URL, retrying when the CDN refuses.
 *
 * Rate limiting is the normal failure here, not the exception: a full run is 270 requests and Microsoft
 * answers 429 for minutes afterwards.  A 429 carries Retry-After often enough to be worth reading, and
 * waiting what the server asks for beats guessing -- whereas any other non-2xx is likely permanent for
 * this URL, so it gets one quick retry rather than the patient treatment.
 *
 * @param  {String} url
 * @param  {Number} attempts
 * @returns {String|Dictionary|undefined}  undefined when every attempt failed
 */
async function fetchFromLearn(url, attempts = 3) {
  for (let attempt = 1; attempt <= attempts; attempt++) {
    // A successful GET always resolves to a string or dictionary, so undefined is a safe "it failed"
    // sentinel and the wait is decided by whichever tolerate handler ran.
    let secondsToWait = 2;
    let responseBody = await sails.helpers.http.get(url)
    .tolerate('non200Response', (serverResponse)=>{
      if(serverResponse.statusCode === 429) {
        let retryAfter = serverResponse.headers ? serverResponse.headers['retry-after'] : undefined;
        secondsToWait = Math.min(Number(retryAfter) || 10, 60);
      }
      return undefined;
    })
    .tolerate(()=>{
      return undefined;
    });

    if(responseBody !== undefined) {
      return responseBody;
    }
    if(attempt < attempts) {
      await sails.helpers.flow.pause(secondsToWait * 1000 * attempt);
    }
  }
  return undefined;
}


/**
 * Parse one rendered Policy CSP area page into its nodes.
 *
 * Each node on these pages is an <h2> whose section carries a "Description framework properties" table
 * (Format, Access Type, Default Value), an optional "Allowed values" table, and an optional dependency
 * block.  Tags are replaced with a separator and the text read as a flat run of cells, because the
 * surrounding markup differs between the table and definition-list renderings of the same facts while
 * the cell order does not.
 *
 * @param  {String} pageHtml
 * @returns {Array} nodes
 */
function parseAreaPage(pageHtml) {

  const SKIPPED_HEADINGS = ['In this article', 'Related articles', 'Feedback', 'Additional resources'];
  // Deliberately narrow: an allowed-value cell is a number, a hex literal, or a short token, optionally
  // marked as the default.  Anything longer is prose belonging to the value above it.
  const LOOKS_LIKE_A_VALUE_CELL = /^[0-9A-Fa-fxX.-]{1,12}( \(Default\))?$/;

  let stripTags = (fragment)=>{
    let withoutScripts = fragment;
    let previous;
    do {
      previous = withoutScripts;
      withoutScripts = withoutScripts.replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/gi, '');
    } while (withoutScripts !== previous);
    return _.filter(
      _.map(withoutScripts.replace(/<[^>]+>/g, '\u0000').split('\u0000'), (cell)=>{ return cell.trim(); }),
      (cell)=>{ return cell !== ''; }
    );
  };
  // The rendered pages are the only source available, so entities have to be decoded by hand rather than
  // by whatever parsed the markdown.  Only the five that XML defines plus the numeric forms appear here.
  let decodeEntities = (str)=>{
    return str
      .replace(/&#(\d+);/g, (unusedMatch, code)=>{ return String.fromCharCode(Number(code)); })
      .replace(/&#x([0-9a-fA-F]+);/g, (unusedMatch, code)=>{ return String.fromCharCode(parseInt(code, 16)); })
      .replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"')
      .replace(/&#39;|&apos;/g, '\'').replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&');
  };

  // Most area pages put one node per <h2>, but a couple -- Update, LocalUsersAndGroups -- group their
  // nodes into <h2> categories and put the nodes themselves at <h3>.  Both levels are collected and a
  // section runs to the next heading of either level, so a category's own section ends before the first
  // node under it rather than swallowing that node's properties table.  Update alone has 94 nodes that
  // an <h2>-only parser misses entirely.
  let nodes = [];
  let headings = [];
  let headingRegExp = /<h([23])[^>]*id="([^"]+)"[^>]*>([\s\S]*?)<\/h\1>/g;
  let headingMatch;
  while ((headingMatch = headingRegExp.exec(pageHtml)) !== null) {
    headings.push({name: headingMatch[3].replace(/[<>]/g, '').trim(), endsAt: headingRegExp.lastIndex, startsAt: headingMatch.index});
  }

  for (let idx = 0; idx < headings.length; idx++) {
    let heading = headings[idx];
    if(_.contains(SKIPPED_HEADINGS, heading.name)) {
      continue;
    }
    let sectionHtml = pageHtml.slice(heading.endsAt, idx + 1 < headings.length ? headings[idx + 1].startsAt : pageHtml.length);
    let cells = _.map(stripTags(sectionHtml), decodeEntities);

    // Property labels render bare in tables and with a trailing colon in definition lists, and the colon
    // sometimes lands in a cell of its own -- so match either and step over it.
    let propertyNamed = (label)=>{
      for (let cellIdx = 0; cellIdx < cells.length; cellIdx++) {
        if(cells[cellIdx].replace(/:$/, '') === label) {
          for (let candidate of cells.slice(cellIdx + 1, cellIdx + 3)) {
            if(candidate !== ':') {
              return candidate;
            }
          }
        }
      }
      return undefined;
    };

    let locUris = _.uniq(sectionHtml.match(/\.\/(?:Device|User)\/Vendor\/MSFT\/Policy\/Config\/[A-Za-z0-9_\-/]+/g) || []);
    let deviceScoped = _.filter(locUris, (locUri)=>{ return _.startsWith(locUri, './Device/'); });
    let userScoped = _.filter(locUris, (locUri)=>{ return _.startsWith(locUri, './User/'); });
    // A node published in both scopes lists both paths.  Device is what an MDM-delivered profile targets,
    // so it wins; a User-only node keeps its own path and is flagged, since writing it under ./Device/
    // deploys cleanly and enforces nothing.
    let locUri = _.first(deviceScoped) || _.first(userScoped);
    if(!locUri) {
      continue;
    }

    // A value's description can run to several paragraphs and pick up trailing notes, so cells are
    // attributed to the value above them rather than read at a fixed stride.
    let allowedValues = [];
    let allowedValuesAt = _.indexOf(cells, 'Allowed values');
    if(allowedValuesAt !== -1) {
      let afterLabel = cells.slice(allowedValuesAt);
      let descriptionAt = _.indexOf(afterLabel.slice(0, 6), 'Description');
      if(descriptionAt !== -1) {
        let current;
        for (let cell of afterLabel.slice(descriptionAt + 1)) {
          if(_.contains(['Group policy mapping', 'Description framework properties', 'Note'], cell)) {
            break;
          }
          if(LOOKS_LIKE_A_VALUE_CELL.test(cell)) {
            current = {value: cell.replace(' (Default)', ''), isDefault: _.contains(cell, '(Default)'), description: []};
            allowedValues.push(current);
          } else if(current) {
            current.description.push(cell);
          } else {
            break;
          }
        }
      }
    }

    let dependsOnUri = propertyNamed('Dependency URI');
    nodes.push({
      area: locUri.split('/')[6],
      name: heading.name,
      locUri,
      scopes: (deviceScoped.length > 0 ? ['Device'] : []).concat(userScoped.length > 0 ? ['User'] : []),
      format: propertyNamed('Format'),
      accessType: propertyNamed('Access Type'),
      defaultValue: propertyNamed('Default Value'),
      allowedValues: _.map(allowedValues, (allowedValue)=>{
        return {value: allowedValue.value, isDefault: allowedValue.isDefault, description: allowedValue.description.join(' ')};
      }),
      dependsOn: dependsOnUri ? {locUri: dependsOnUri, allowedValue: propertyNamed('Dependency Allowed Value')} : undefined,
      // Ten nodes carry this, all of them passcode policy.  Getting it wrong is a delivery failure rather
      // than a silent no-op, which makes it worth the byte it costs.
      mustBeWrappedInAtomic: _.contains(sectionHtml, 'must be wrapped in an Atomic command'),
    });
  }

  return nodes;
}
