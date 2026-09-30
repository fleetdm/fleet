module.exports = {


  friendlyName: 'Regenerate Windows CSP nodes',


  description: 'Scrape Microsoft\'s published CSP reference and save every node\'s LocURI, format, access type, and allowed values to website/profile-generator/schema/windows-csp-nodes.json.',


  extendedDescription:
`The configuration profile generator has an embedded schema for .mobileconfig payloads and DDM
declarations, but until now nothing for Windows CSP -- it was given a documentation URL and asked to
recall node paths from memory.  It recalled them wrong: the wrong area segment (System/AllowClipboardHistory
for Experience/AllowClipboardHistory), a truncated prefix (./DeviceLock/DevicePasswordEnabled), and
inverted booleans on nodes where 0 means enabled.

Microsoft publishes the format, access type, default, allowed values and dependencies for every node on
the per-area Policy CSP pages, so this walks all of them and writes the result out as data.  The scrape
is the only way in: the markdown behind those pages lives in MicrosoftDocs/windows-docs-pr, which is private.

Every other CSP (WiFi, Firewall, BitLocker, VPNv2, ...) is read from its "DDF file" page instead of its
reference page.  About a third of those reference pages predate the generated format and are prose that
names a node by its leaf alone, while the DDF page embeds the device description framework XML that the
node tree is defined by, so paths come out exact for all of them.

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

    // Every node the profile generator's own test cases depend on, plus one from each shape of DDF page
    // (TenantLockdown's has no MgmtTree wrapper, WiFi's has dynamic nodes in both scopes).  These are the
    // assertions that were failing before this file existed, so if the parser stops finding any of them it
    // has regressed in exactly the way that matters, whatever the total node count says.
    let NODES_THAT_MUST_PARSE = {
      './Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxDevicePasswordFailedAttempts': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/Storage/RemovableDiskDenyWriteAccess': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/Experience/AllowClipboardHistory': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/System/AllowTelemetry': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowAppStoreAutoUpdate': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/WindowsAI/DisableAIDataAnalysis': {format: 'int'},
      './Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/InteractiveLogon_MessageTextForUsersAttemptingToLogOn': {format: 'chr'},
      './Vendor/MSFT/Firewall/MdmStore/DomainProfile/EnableFirewall': {format: 'bool'},
      './Vendor/MSFT/Firewall/MdmStore/DomainProfile/AllowLocalPolicyMerge': {format: 'bool'},
      './Device/Vendor/MSFT/WiFi/Profile/{SSID}/WlanXml': {format: 'chr'},
      './User/Vendor/MSFT/WiFi/Profile/{SSID}/WlanXml': {format: 'chr'},
      './Device/Vendor/MSFT/BitLocker/RequireDeviceEncryption': {format: 'int'},
      './Device/Vendor/MSFT/VPNv2/{ProfileName}/ProfileXML': {format: 'chr'},
      './Vendor/MSFT/TenantLockdown/RequireNetworkInOOBE': {format: 'bool'},
      './Device/Vendor/MSFT/Defender/Configuration/HideExclusionsFromLocalUsers': {format: 'int'},
      './Device/Vendor/MSFT/Defender/Configuration/EnableFileHashComputation': {format: 'int'},
      './Vendor/MSFT/Firewall/MdmStore/DomainProfile/DefaultInboundAction': {format: 'int'},
      './Vendor/MSFT/Firewall/MdmStore/PublicProfile/AllowLocalIpsecPolicyMerge': {format: 'bool'},
      './Device/Vendor/MSFT/PassportForWork/Biometrics/FacialFeaturesUseEnhancedAntiSpoofing': {format: 'bool'},
    };

    // A scrape that half-works is worse than one that fails, because it writes a file that looks fine and
    // is missing a third of the surface.  Sized off the current published set (269 Policy areas, ~3,200
    // Policy nodes; 77 other CSPs, ~2,500 nodes) with enough headroom that Microsoft adding or retiring
    // policies does not trip it.
    let MINIMUM_PLAUSIBLE_AREA_COUNT = 200;
    let MINIMUM_PLAUSIBLE_NODE_COUNT = 2500;
    let MINIMUM_PLAUSIBLE_OTHER_CSP_COUNT = 60;
    let MINIMUM_PLAUSIBLE_OTHER_CSP_NODE_COUNT = 1800;

    let tableOfContents = await getTableOfContents(LEARN_BASE_URL);
    let areaSlugs = getPolicyAreaSlugs(tableOfContents);
    let ddfSlugsByCspSlug = getDdfSlugsByCspSlug(tableOfContents);
    sails.log(`Found ${areaSlugs.length} Policy CSP area pages and ${Object.keys(ddfSlugsByCspSlug).length} other CSPs.`);

    // Microsoft's CDN throttles a sustained burst rather than a fast one: at eight concurrent it starts
    // erroring a third of the way into a 269-page run, and even at four the last twenty areas fail.  So
    // the first pass goes wide and a second pass picks up whatever it dropped, one at a time with long
    // waits.  A dropped page is indistinguishable from an area with no nodes once parsed, which is why
    // this has to be reliable here rather than checked downstream.
    let nodes = [];
    let otherCspNodes = [];
    let cspsWithoutNodeReference = [];
    let CONCURRENCY = 4;

    let fetchArea = async (areaSlug)=>{
      let pageHtml = await fetchFromLearn(`${LEARN_BASE_URL}/policy-csp-${areaSlug}`);
      if(!pageHtml) {
        return false;
      }
      nodes = nodes.concat(parseAreaPage(pageHtml));
      return true;
    };

    let fetchDdfPage = async (ddfSlug)=>{
      let pageHtml = await fetchFromLearn(`${LEARN_BASE_URL}/${ddfSlug}`);
      if(!pageHtml) {
        return false;
      }
      let parsed = parseDdfPage(pageHtml);
      if(parsed.deprecatedInFavorOf !== undefined) {
        // DeviceLock, Storage and VPN: each superseded wholesale (by Policy and VPNv2), and each with nodes
        // named exactly like the live ones that replaced them, so listing them would hand the generator a
        // second, dead DevicePasswordEnabled to choose.
        cspsWithoutNodeReference.push({page: ddfSlug, reason: `deprecated -- use ${parsed.deprecatedInFavorOf}`});
      } else if(parsed.nodes.length === 0) {
        cspsWithoutNodeReference.push({page: ddfSlug, reason: 'DDF page has no parseable node tree'});
      }
      otherCspNodes = otherCspNodes.concat(parsed.nodes);
      return true;
    };

    let pagesToFetch = _.map(areaSlugs, (areaSlug)=>{ return {label: `policy-csp-${areaSlug}`, fetch: ()=>fetchArea(areaSlug)}; });
    for (let cspSlug of Object.keys(ddfSlugsByCspSlug).sort()) {
      let ddfSlugs = ddfSlugsByCspSlug[cspSlug];
      if(ddfSlugs.length === 0) {
        // Mostly the OMA Client Provisioning CSPs (NAP, PXLOGICAL, w4 APPLICATION, ...), whose pages are
        // prose with no DDF behind them.  Recorded rather than dropped so their absence reads as known.
        cspsWithoutNodeReference.push({page: cspSlug, reason: 'no DDF page published'});
      }
      for (let ddfSlug of ddfSlugs) {
        pagesToFetch.push({label: ddfSlug, fetch: ()=>fetchDdfPage(ddfSlug)});
      }
    }

    let pagesToRetry = [];
    let pagesFetchedSoFar = 0;
    for (let batch of _.chunk(pagesToFetch, CONCURRENCY)) {
      await sails.helpers.flow.simultaneouslyForEach(batch, async (page)=>{
        if(!await page.fetch()) {
          pagesToRetry.push(page);
        }
      });
      pagesFetchedSoFar += batch.length;
      if(pagesFetchedSoFar % 40 < CONCURRENCY || pagesFetchedSoFar === pagesToFetch.length) {
        sails.log(`  ...${pagesFetchedSoFar}/${pagesToFetch.length} pages`);
      }
    }

    let pagesThatFailed = [];
    if(pagesToRetry.length > 0) {
      sails.log(`Retrying ${pagesToRetry.length} page(s) the first pass could not fetch, one at a time...`);
      for (let page of pagesToRetry) {
        let fetched = false;
        for (let attempt = 1; attempt <= 5 && !fetched; attempt++) {
          await sails.helpers.flow.pause(2000 * attempt);
          fetched = await page.fetch();
        }
        if(!fetched) {
          pagesThatFailed.push(page.label);
        }
      }
    }

    // Hard failure, not a warning.  A page that never arrived parses as zero nodes, so letting the run
    // continue writes a file that is quietly missing a chunk of the CSP surface.
    if(pagesThatFailed.length > 0) {
      throw new Error(
        `Refusing to overwrite the committed node file: ${pagesThatFailed.length} page(s) could not be ` +
        `fetched, even after retrying them individually:\n  ${pagesThatFailed.join('\n  ')}\n\n` +
        `If those say 429, this is rate limiting rather than anything wrong with the script -- one full run ` +
        `is ~350 requests, and Microsoft starts refusing for several minutes after a couple of runs ` +
        `back to back.  Wait, then run it again.`
      );
    }

    // Every real node documents a format.  A section with a path but no format is prose that happens to
    // mention a LocURI -- the UserRights page's "General example" heading, for one -- so the format is
    // what separates a policy from a worked example, not the path.
    let nodesWithAPath = _.filter(nodes, (node)=>{ return node.locUri && node.format; });
    let areaNames = _.uniq(_.pluck(nodesWithAPath, 'area'));
    sails.log(`Parsed ${nodesWithAPath.length} Policy CSP nodes across ${areaNames.length} areas.`);

    // A few DDF pages embed the same tree more than once (EnterpriseAPN has one per OS release), and
    // the later copy is the current one.
    let otherCspNodesByLocUri = {};
    for (let node of otherCspNodes) {
      otherCspNodesByLocUri[node.locUri] = node;
    }
    let otherCspNodesWithAPath = _.filter(_.values(otherCspNodesByLocUri), (node)=>{ return node.format; });
    let otherCspNames = _.uniq(_.pluck(otherCspNodesWithAPath, 'csp'));
    sails.log(`Parsed ${otherCspNodesWithAPath.length} nodes across ${otherCspNames.length} other CSPs.`);
    for (let skipped of cspsWithoutNodeReference) {
      sails.log(`  Skipped ${skipped.page}: ${skipped.reason}`);
    }

    let allNodes = nodesWithAPath.concat(otherCspNodesWithAPath);
    let missing = [];
    for (let expectedLocUri of Object.keys(NODES_THAT_MUST_PARSE)) {
      let found = _.find(allNodes, {locUri: expectedLocUri});
      if(!found) {
        missing.push(`${expectedLocUri} -- not found at all`);
      } else if(found.format !== NODES_THAT_MUST_PARSE[expectedLocUri].format) {
        missing.push(`${expectedLocUri} -- parsed format "${found.format}", expected "${NODES_THAT_MUST_PARSE[expectedLocUri].format}"`);
      }
    }

    if(areaNames.length < MINIMUM_PLAUSIBLE_AREA_COUNT || nodesWithAPath.length < MINIMUM_PLAUSIBLE_NODE_COUNT) {
      throw new Error(
        `Refusing to overwrite the committed node file: this run parsed ${nodesWithAPath.length} Policy CSP ` +
        `nodes across ${areaNames.length} areas, below the ${MINIMUM_PLAUSIBLE_NODE_COUNT}/${MINIMUM_PLAUSIBLE_AREA_COUNT} floor.  ` +
        `Microsoft has most likely changed the markup these pages are built from, so the parser needs updating ` +
        `rather than the data.`
      );
    }
    if(otherCspNames.length < MINIMUM_PLAUSIBLE_OTHER_CSP_COUNT || otherCspNodesWithAPath.length < MINIMUM_PLAUSIBLE_OTHER_CSP_NODE_COUNT) {
      throw new Error(
        `Refusing to overwrite the committed node file: this run parsed ${otherCspNodesWithAPath.length} nodes ` +
        `across ${otherCspNames.length} non-Policy CSPs, below the ${MINIMUM_PLAUSIBLE_OTHER_CSP_NODE_COUNT}/${MINIMUM_PLAUSIBLE_OTHER_CSP_COUNT} floor.  ` +
        `Microsoft has most likely changed how DDF pages embed their XML, so the parser needs updating rather ` +
        `than the data.`
      );
    }
    if(missing.length > 0) {
      throw new Error(
        `Refusing to overwrite the committed node file: ${missing.length} node(s) the profile generator's ` +
        `test cases depend on did not parse as expected:\n  ${missing.join('\n  ')}`
      );
    }

    let outputPath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-csp-nodes.json');
    if(dry) {
      sails.log(`Dry run -- not writing.  Would have written ${allNodes.length} nodes to ${outputPath}.`);
      return;
    }

    // Sorted so a regeneration produces a readable diff instead of reshuffling thousands of lines whenever
    // Microsoft reorders a page.
    let sortedNodes = _.sortBy(allNodes, (node)=>{ return node.csp === 'Policy' ? `Policy/${node.area}/${node.name}` : `${node.csp}/${node.locUri}`; });
    await sails.helpers.fs.writeJson.with({
      destination: outputPath,
      json: {
        generatedAt: (new Date()).toISOString(),
        source: `${LEARN_BASE_URL}/`,
        cspsWithoutNodeReference: _.sortBy(cspsWithoutNodeReference, 'page'),
        nodes: sortedNodes,
      },
      force: true
    });

    sails.log(`\nWrote ${sortedNodes.length} CSP nodes to ${outputPath}.`);
    sails.log(`Read the diff before committing -- this file is scraped from rendered HTML, so a markup change shows up as content.`);

  }


};


/**
 * Fetch the MDM table of contents, which is the only index of the CSP and Policy CSP area pages.
 *
 * @param  {String} learnBaseUrl
 * @returns {Dictionary}
 */
async function getTableOfContents(learnBaseUrl) {

  // Retried like the other pages are.  This is one request out of ~350, but it is the one that decides
  // whether the run happens at all, and losing it to a throttled moment fails the script with a bare
  // "non200Response" rather than anything a reader could act on.
  let tableOfContents = await fetchFromLearn(`${learnBaseUrl}/toc.json`, 5);
  if(!tableOfContents) {
    throw new Error(
      `Could not fetch the MDM table of contents at ${learnBaseUrl}/toc.json, so there is no list of ` +
      `CSP pages to walk.  A 429 here means rate limiting rather than anything wrong with the ` +
      `script: a full run is ~350 requests and Microsoft refuses for several minutes after a couple of ` +
      `runs back to back.  Wait, then try again.`
    );
  }
  return tableOfContents;
}


/**
 * Get the slug of every Policy CSP area page from the MDM table of contents.
 *
 * @param  {Dictionary} tableOfContents
 * @returns {Array} area slugs, e.g. ['devicelock', 'storage', ...]
 */
function getPolicyAreaSlugs(tableOfContents) {

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
 * Map every CSP reference page other than Policy's to the DDF page(s) the table of contents nests under it.
 *
 * @param  {Dictionary} tableOfContents
 * @returns {Dictionary} e.g. {'wifi-csp': ['wifi-ddf-file'], 'nap-csp': [], ...}
 */
function getDdfSlugsByCspSlug(tableOfContents) {

  // DDF slugs are not derivable from the CSP's (wifi-ddf-file, defender-ddf, applicationcontrol-csp-ddf
  // all occur), so they are read off the entry's children.  A CSP can appear more than once -- DMClient
  // and DeclaredConfiguration are also listed, childless, under "Declared Configuration" -- so children
  // are merged across every appearance rather than taken from the first.
  let ddfSlugsByCspSlug = {};
  let walk = (branch)=>{
    if(_.isArray(branch)) {
      _.each(branch, walk);
    } else if(_.isObject(branch)) {
      if(_.isString(branch.href) && /^[a-z0-9-]+-csp$/.test(branch.href) && !_.startsWith(branch.href, 'policy-csp-')) {
        let childHrefs = _.filter(_.pluck(_.isArray(branch.children) ? branch.children : [], 'href'), (href)=>{
          return _.isString(href) && _.contains(href, 'ddf');
        });
        ddfSlugsByCspSlug[branch.href] = _.uniq((ddfSlugsByCspSlug[branch.href] || []).concat(childHrefs));
      }
      _.each(_.values(branch), walk);
    }
  };
  walk(tableOfContents);
  return ddfSlugsByCspSlug;
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

    // The prose between the node's LocURI and its properties table says what the node does.  Without it
    // the generator chose between lookalikes by name alone -- NotifyPasswordReuse for a malicious-site
    // warning, since both it and NotifyMalicious only document "0=Disabled 1=Enabled".
    let lastLocUriCellAt = _.findLastIndex(cells, (cell)=>{ return /^\.\/(?:Device|User)\/Vendor\/MSFT\/Policy\/Config\//.test(cell); });
    let propertiesTableAt = _.indexOf(cells, 'Description framework properties');
    let descriptionCells = (lastLocUriCellAt !== -1 && propertiesTableAt > lastLocUriCellAt) ? cells.slice(lastLocUriCellAt + 1, propertiesTableAt) : [];

    // An int node's bounds sit in the properties table as "Allowed Values | Range: | [4-16]" -- capital V,
    // unlike the "Allowed values" heading of the enum table above -- and the label and the bracket are
    // usually separate cells, so the next few cells are read together.
    let allowedValuesPropertyAt = _.findIndex(cells, (cell)=>{ return cell.replace(/:$/, '') === 'Allowed Values'; });
    let rangeMatch = allowedValuesPropertyAt === -1 ? null : cells.slice(allowedValuesPropertyAt + 1, allowedValuesPropertyAt + 4).join(' ').match(/Range:\s*(\[[^\]]+\])/);

    let dependsOnUri = propertyNamed('Dependency URI');
    nodes.push({
      csp: 'Policy',
      area: locUri.split('/')[6],
      name: heading.name,
      locUri,
      description: summarizeDescription(descriptionCells.join(' ')),
      scopes: (deviceScoped.length > 0 ? ['Device'] : []).concat(userScoped.length > 0 ? ['User'] : []),
      format: propertyNamed('Format'),
      accessType: propertyNamed('Access Type'),
      defaultValue: propertyNamed('Default Value'),
      allowedValues: _.map(allowedValues, (allowedValue)=>{
        return {value: allowedValue.value, isDefault: allowedValue.isDefault, description: allowedValue.description.join(' ')};
      }),
      allowedRange: rangeMatch ? rangeMatch[1] : undefined,
      dependsOn: dependsOnUri ? {locUri: dependsOnUri, allowedValue: propertyNamed('Dependency Allowed Value')} : undefined,
      // Ten nodes carry this, all of them passcode policy.  Getting it wrong is a delivery failure rather
      // than a silent no-op, which makes it worth the byte it costs.
      mustBeWrappedInAtomic: _.contains(sectionHtml, 'must be wrapped in an Atomic command'),
    });
  }

  return nodes;
}


/**
 * Keep the leading sentences of a node's description that fit in 200 characters.
 *
 * Whole sentences only, dropping the first that does not fit rather than cutting it: a description cut
 * mid-sentence can say something the full text does not, which is what happened to the Apple schema's
 * minLength ("independent of the value for…").  Only a first sentence too long on its own is cut.
 *
 * @param  {String} text
 * @returns {String|undefined}
 */
function summarizeDescription(text) {
  const MAX_LENGTH = 200;
  let squashed = String(text || '').replace(/\s+/g, ' ').trim();
  if(!squashed) {
    return undefined;
  }
  let sentences = squashed.split(/(?<=[a-z0-9)]\.)\s+(?=[A-Z])/);
  let kept = [_.first(sentences)];
  for (let sentence of _.rest(sentences)) {
    if(kept.concat(sentence).join(' ').length > MAX_LENGTH) {
      break;
    }
    kept.push(sentence);
  }
  let summary = kept.join(' ');
  return summary.length <= MAX_LENGTH ? summary : summary.slice(0, MAX_LENGTH).replace(/\s+\S*$/, '') + '…';
}


/**
 * Decode the entities found in rendered Learn pages.
 *
 * The rendered pages are the only source available, so entities have to be decoded by hand rather than
 * by whatever parsed the markdown.  Only the five that XML defines plus the numeric forms appear here.
 *
 * @param  {String} str
 * @returns {String}
 */
function decodeEntities(str) {
  return str
    .replace(/&#(\d+);/g, (unusedMatch, code)=>{ return String.fromCharCode(Number(code)); })
    .replace(/&#x([0-9a-fA-F]+);/g, (unusedMatch, code)=>{ return String.fromCharCode(parseInt(code, 16)); })
    .replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"')
    .replace(/&#39;|&apos;/g, '\'').replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&');
}


/**
 * Parse one rendered CSP "DDF file" page into its nodes.
 *
 * The page embeds the CSP's device description framework: a tree of <Node> elements, each with a
 * <NodeName>, an optional <Path> (on the root only), and a <DFProperties> block holding its access type,
 * format, default and, on the newer DDFs, MSFT:AllowedValues and MSFT:DependencyBehavior.  A node's LocURI
 * is its ancestors' names joined onto the root's <Path>.  <DFProperties> never contains a <Node>, so the
 * tree is walked as a flat run of open, close, name, path and properties tokens rather than parsed as XML
 * -- the website has no XML parser as a direct dependency, and a couple of these pages ship fragments
 * that are not well-formed documents anyway (WindowsAutopilot's opens at a bare <NodeName>).
 *
 * @param  {String} pageHtml
 * @returns {Dictionary} {nodes, deprecatedInFavorOf}
 */
function parseDdfPage(pageHtml) {

  let mainHtml = pageHtml.slice(Math.max(0, pageHtml.indexOf('<main')));

  // A deprecated CSP's DDF page opens with the notice and then embeds its XML unescaped, which the page
  // renders as a run of bare text.  Nothing below would parse it anyway, but reporting it by name keeps a
  // deprecation from reading as a parser regression.
  let beforeFirstCodeBlock = mainHtml.indexOf('<pre') === -1 ? mainHtml : mainHtml.slice(0, mainHtml.indexOf('<pre'));
  let deprecationMatch = beforeFirstCodeBlock.match(/(?:CSP|configuration service provider|policy) is deprecated\.\s*Use\s*([\s\S]{0,200}?)\s*instead/i);
  if(deprecationMatch) {
    return {nodes: [], deprecatedInFavorOf: decodeEntities(deprecationMatch[1].replace(/<[^>]+>/g, '')).replace(/\s+/g, ' ').trim()};
  }

  let xmlBlocks = [];
  let codeBlockRegExp = /<pre[^>]*>\s*<code[^>]*>([\s\S]*?)<\/code>\s*<\/pre>/g;
  let codeBlockMatch;
  while ((codeBlockMatch = codeBlockRegExp.exec(mainHtml)) !== null) {
    let xml = decodeEntities(codeBlockMatch[1]);
    if(_.contains(xml, '<NodeName')) {
      xmlBlocks.push(xml);
    }
  }

  // Text inside the XML is XML-escaped in its own right, underneath the HTML escaping that was just
  // removed, so values and descriptions are decoded a second time on the way out.
  let textOf = (fragment)=>{
    return fragment === undefined ? undefined : decodeEntities(fragment.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, '$1')).replace(/\s+/g, ' ').trim();
  };
  let firstMatch = (fragment, regExp)=>{
    let match = fragment.match(regExp);
    return match ? match[1] : undefined;
  };

  let nodes = [];
  for (let xml of xmlBlocks) {
    let stack = [];
    let tokenRegExp = /<Node>|<\/Node>|<NodeName\s*\/>|<NodeName>([\s\S]*?)<\/NodeName>|<Path>([\s\S]*?)<\/Path>|<DFProperties>([\s\S]*?)<\/DFProperties>/g;
    let token;
    while ((token = tokenRegExp.exec(xml)) !== null) {
      if(token[0] === '<Node>') {
        stack.push({});
      } else if(token[0] === '</Node>') {
        stack.pop();
      } else if(_.startsWith(token[0], '<NodeName')) {
        if(stack.length === 0) {
          stack.push({});
        }
        _.last(stack).name = (token[1] || '').trim();
      } else if(_.startsWith(token[0], '<Path>')) {
        if(stack.length > 0) {
          _.last(stack).path = token[2].trim();
        }
      } else if(stack.length > 0) {
        let current = _.last(stack);
        let parent = stack.length > 1 ? stack[stack.length - 2] : undefined;
        let properties = token[3];

        // A dynamic node -- one instance per SSID, per VPN profile, per firewall rule -- has an empty
        // <NodeName> and names its placeholder in <DFTitle>.  Microsoft's reference pages write the
        // placeholder in braces, and so does everything that reads this file.
        let nodeName = current.name;
        if(!nodeName) {
          nodeName = `{${textOf(firstMatch(properties, /<DFTitle>([\s\S]*?)<\/DFTitle>/)) || 'NodeName'}}`;
        }
        let parentLocUri = current.path !== undefined ? current.path : (parent ? parent.locUri : undefined);
        if(parentLocUri === undefined) {
          continue;
        }
        current.locUri = `${parentLocUri.replace(/\/$/, '')}/${nodeName}`;
        current.rootLocUri = parent ? parent.rootLocUri : current.locUri;
        current.csp = parent ? parent.csp : nodeName;

        let allowedValues = [];
        let allowedRange;
        let allowedValuesMatch = properties.match(/<MSFT:AllowedValues\s+ValueType="([^"]+)"[^>]*>([\s\S]*?)<\/MSFT:AllowedValues>/);
        let defaultValue = textOf(firstMatch(properties, /<DefaultValue>([\s\S]*?)<\/DefaultValue>/));
        if(allowedValuesMatch && allowedValuesMatch[1] === 'ENUM') {
          let enumRegExp = /<MSFT:Enum>([\s\S]*?)<\/MSFT:Enum>/g;
          let enumMatch;
          while ((enumMatch = enumRegExp.exec(allowedValuesMatch[2])) !== null) {
            let value = textOf(firstMatch(enumMatch[1], /<MSFT:Value>([\s\S]*?)<\/MSFT:Value>/));
            allowedValues.push({
              value,
              isDefault: value !== undefined && value === defaultValue,
              description: textOf(firstMatch(enumMatch[1], /<MSFT:ValueDescription>([\s\S]*?)<\/MSFT:ValueDescription>/)) || '',
            });
          }
        } else if(allowedValuesMatch && allowedValuesMatch[1] === 'Range') {
          allowedRange = textOf(firstMatch(allowedValuesMatch[2], /<MSFT:Value>([\s\S]*?)<\/MSFT:Value>/));
        }

        let accessTypeFragment = firstMatch(properties, /<AccessType>([\s\S]*?)<\/AccessType>/) || '';
        let accessTypes = _.uniq(_.map(accessTypeFragment.match(/<(Add|Delete|Exec|Get|Replace)\s*\/>/g) || [], (tag)=>{ return tag.replace(/[<>/\s]/g, ''); })).sort();

        let dependsOnUri = textOf(firstMatch(properties, /<MSFT:DependencyUri>([\s\S]*?)<\/MSFT:DependencyUri>/));
        // An ENUM-typed dependency wraps its value in the same <MSFT:Enum> shape as AllowedValues does.
        let dependencyAllowedValue = firstMatch(properties, /<MSFT:DependencyAllowedValue[^>]*>([\s\S]*?)<\/MSFT:DependencyAllowedValue>/);
        if(dependencyAllowedValue !== undefined && _.contains(dependencyAllowedValue, '<MSFT:Value>')) {
          dependencyAllowedValue = firstMatch(dependencyAllowedValue, /<MSFT:Value>([\s\S]*?)<\/MSFT:Value>/);
        }
        // An ADMX-typed one holds an <MSFT:AdmxBacked> reference rather than a value, so it records only
        // that the dependency exists.
        dependencyAllowedValue = textOf(dependencyAllowedValue);
        if(dependencyAllowedValue === '' || _.startsWith(dependencyAllowedValue, '<')) {
          dependencyAllowedValue = undefined;
        }
        let scopeSegment = current.rootLocUri.split('/')[1];
        nodes.push({
          csp: current.csp,
          name: current.locUri === current.rootLocUri ? current.csp : current.locUri.slice(current.rootLocUri.length + 1),
          locUri: current.locUri,
          description: summarizeDescription(textOf(firstMatch(properties, /<Description>([\s\S]*?)<\/Description>/))),
          // A root at ./Vendor/MSFT (Firewall, TenantLockdown, UEFI, ...) predates the scoped paths and is
          // device-wide.
          scopes: scopeSegment === 'User' ? ['User'] : ['Device'],
          format: firstMatch(properties, /<DFFormat>\s*<([A-Za-z0-9]+)\s*\/>/),
          accessType: accessTypes.length > 0 ? accessTypes.join(', ') : undefined,
          defaultValue,
          allowedValues,
          allowedRange,
          dependsOn: dependsOnUri ? {locUri: dependsOnUri, allowedValue: dependencyAllowedValue} : undefined,
          mustBeWrappedInAtomic: /<MSFT:AtomicRequired\s*\/>/.test(properties),
          deprecated: /<MSFT:Deprecated\b/.test(properties) || undefined,
        });
      }
    }
  }

  return {nodes, deprecatedInFavorOf: undefined};
}
