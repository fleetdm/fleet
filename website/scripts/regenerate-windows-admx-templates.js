module.exports = {


  friendlyName: 'Regenerate Windows ADMX templates',


  description: 'Download the third-party ADMX templates the profile generator supports, save them to website/profile-generator/schema/admx-templates/, and write every policy they define to website/profile-generator/schema/windows-admx-templates.json.',


  extendedDescription:
`Third-party Windows app settings (Chrome, Firefox, ...) are not part of the Policy CSP.  They become CSP
nodes only after the app's ADMX file is ingested through
./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/{AppName}/Policy/{AdmxFileName}, and their LocURIs
are then built from that AppName and the ADMX file's category tree.  Nothing in windows-csp-nodes.json covers
them, so the generator had to recall paths and <data id>s from memory.

This downloads each vendor's published package, saves the .admx it ships (as UTF-8, since Google ships
UTF-16 and the file is embedded verbatim in a UTF-8 profile), and parses it -- with the en-US .adml for
display names and descriptions -- into nodes shaped like windows-csp-nodes.json's, plus the per-element
detail an ADMX-backed policy's <data> payload needs.  The .adml is only read, never saved: Windows does not
need it to ingest a template.

Needs \`unzip\` on the PATH.  Read the diff before committing: a vendor reorganizing its categories changes
every LocURI under them.`,


  inputs: {

    dry: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Download and parse, but do not write anything.'
    },

  },


  fn: async function ({dry}) {

    let path = require('path');
    let fs = require('fs');
    let os = require('os');
    let childProcess = require('child_process');

    let SCHEMA_DIR = path.resolve(sails.config.appPath, 'profile-generator/schema');

    // One entry per app.  Everything else about a template is worked out from the vendor's package: its id
    // (the display name in kebab case), where it is saved (admx-templates/<id>/<file name>.admx), its AppName
    // (the namespace its .admx defines, without "Policies", e.g., Google.Policies.Chrome is GoogleChrome), where
    // it is ingested (ADMXInstall/<AppName>/Policy/<AppName>AdmxFile), its en-US .adml
    // (<folder>/en-US/<name>.adml, the layout every vendor ships).

    // To add an ADMX template, add an entry to this list following the commented-out example below, then run `sails run regenerate-windows-admx-templates`.
    let TEMPLATES = [
      /*
        {
          // The app's name as admins know it, usually the vendor's product name, e.g., 'Google Chrome'.  Its kebab
          // case form becomes the template's id and folder name (admx-templates/google-chrome/).
          // Affects generated profiles: it labels the template's policies for the generator's lookup, e.g.,
          // "(Google Chrome template)", and names the template in delivery notes.
          displayName: 'Contoso App',

          // The words an admin would use for the app in a request, lowercase, e.g., ['chrome'].  Include the
          // product's common short names; matched as whole words, so 'chrome' does not match 'chromebook'.
          // Affects generated profiles: the template's policies are only offered to a request that mentions one
          // of these.
          keywords: ['contoso'],

          // 'url' when the vendor publishes a fixed download link for its templates, or 'github release' when they
          // are attached to each release of a GitHub repo.
          downloadType: 'url',

          // With downloadType 'url': a direct link to the vendor's .zip of ADMX templates.  Find it on the vendor's
          // enterprise or admin docs, usually a page titled "Group Policy templates" or "ADMX templates".  It must
          // be a .zip, not a web page, .msi, .exe or .cab.
          packageUrl: 'https://downloads.contoso.com/contoso-admx-templates.zip',

          // With downloadType 'github release': the repo, as <owner>/<repo>, from its URL, e.g., 'mozilla/policy-templates'
          // for https://github.com/mozilla/policy-templates.  The script uses the .zip on its latest release.
          repo: 'contoso/policy-templates',

          // Optional, with downloadType 'github release': part of the right .zip's file name, for a release that has
          // more than one .zip.  Look at the files attached to the repo's latest release.  The script stops with an
          // error naming the .zip files if this is needed and missing.
          releaseItemContainsInName: 'Policies',

          // Where the .admx sits inside the .zip.  Download the .zip and list its contents (unzip -l <file>.zip),
          // then copy the path of the app's .admx -- not its .adm or .adml.  The script expects the en-US .adml at
          // <same folder>/en-US/<same name>.adml.
          admxPathInPackage: 'windows/contoso.admx',

          // Optional: leave it out unless the template is mostly policies for other products.  To use it, open the
          // .admx and list the name="..." of each <category> whose policies should be offered.  Policies in other
          // categories are left out of the generator, but the whole file is still installed.
          // Affects generated profiles: only the listed categories' policies are offered.
          onlyCategories: ['Cat_ContosoGeneral'],
        },
      */
      {
        displayName: 'Google Chrome',
        keywords: ['chrome'],
        downloadType: 'url',
        packageUrl: 'https://dl.google.com/dl/edgedl/chrome/policy/policy_templates.zip',
        admxPathInPackage: 'windows/admx/chrome.admx',
      },
      {
        displayName: 'Google Update',
        keywords: ['chrome', 'google update', 'google drive', 'gcpw'],
        downloadType: 'url',
        packageUrl: 'https://dl.google.com/dl/update2/enterprise/googleupdateadmx.zip',
        admxPathInPackage: 'GoogleUpdateAdmx/GoogleUpdate.admx',
        // The same seven update policies for ~50 Google apps, most of them long retired (Gears, O3D, Google
        // Talk).  Indexing all of them would hand the lookup fifty lookalike groups for every "Chrome updates"
        // request, so only the general settings and the apps admins still deploy are kept.
        onlyCategories: ['Cat_GoogleUpdate', 'Cat_Preferences', 'Cat_ProxyServer', 'Cat_Applications', 'Cat_GoogleChrome', 'Cat_GoogleDriveFileStream', 'Cat_GoogleCredentialProviderforWindowsGCPW'],
      },
      {
        displayName: 'Mozilla Firefox',
        keywords: ['firefox', 'mozilla'],
        downloadType: 'github release',
        repo: 'mozilla/policy-templates',
        admxPathInPackage: 'windows/firefox.admx',
      },
    ];

    // Ingested policies may not write under these keys, apart from the exceptions Microsoft lists.  Windows rejects such a policy when the template is ingested, so it is dropped here rather than offered to the generator.
    // [?]: https://learn.microsoft.com/en-us/windows/client-management/win32-and-centennial-app-policy-configuration
    let BLOCKED_REGISTRY_PREFIXES = ['system\\', 'software\\microsoft\\', 'software\\policies\\microsoft\\'];
    let ALLOWED_REGISTRY_PREFIXES = [
      'software\\policies\\microsoft\\office\\', 'software\\microsoft\\office\\', 'software\\microsoft\\windows\\currentversion\\explorer\\',
      'software\\microsoft\\internet explorer\\', 'software\\policies\\microsoft\\shared tools\\proofing tools\\', 'software\\policies\\microsoft\\imejp\\',
      'software\\policies\\microsoft\\ime\\shared\\', 'software\\policies\\microsoft\\shared tools\\graphics filters\\',
      'software\\policies\\microsoft\\windows\\currentversion\\explorer\\', 'software\\policies\\microsoft\\softwareprotectionplatform\\',
      'software\\policies\\microsoft\\officesoftwareprotectionplatform\\', 'software\\policies\\microsoft\\windows\\windows search\\preferences\\',
      'software\\policies\\microsoft\\exchange\\', 'software\\microsoft\\shared tools\\proofing tools\\', 'software\\microsoft\\shared tools\\graphics filters\\',
      'software\\microsoft\\windows\\windows search\\preferences\\', 'software\\microsoft\\exchange\\', 'software\\policies\\microsoft\\vba\\security\\',
      'software\\microsoft\\onedrive', 'software\\microsoft\\edge', 'software\\microsoft\\edgeupdate\\', 'software\\microsoft\\visualstudio', 'software\\policies\\microsoft\\visualstudio',
    ];

    // Make a temporary folder to download each vendor's .zip into; it is deleted when the script finishes.
    let workDir = fs.mkdtempSync(path.join(os.tmpdir(), 'admx-templates-'));

    // Create a helper function to read one file out of a downloaded .zip as text, converting it to UTF-8 with LF line endings.
    let readFromPackage = (zipPath, entryPath)=>{
      let bytes = childProcess.execFileSync('unzip', ['-p', zipPath, entryPath], {maxBuffer: 256 * 1024 * 1024});
      if(bytes.length === 0) {
        throw new Error(`${entryPath} is missing or empty in ${zipPath}.`);
      }
      // Google ships its templates as UTF-16LE with a BOM; Mozilla as UTF-8.
      let text = (bytes[0] === 0xFF && bytes[1] === 0xFE) ? bytes.slice(2).toString('utf16le') : bytes.toString('utf8');
      return text.replace(/^﻿/, '').replace(/\r\n/g, '\n');
    };


    // Now download each listed ADMX template, and parse its policies.
    let downloadedPackages = {};
    let templatesToWrite = [];
    let allNodes = [];
    // Note: we're using a try - finally block so we can delete the temporary folder even if the script throws an error part-way through.
    try {

      for (let listedTemplate of TEMPLATES) {

        // Skip downloading a package that was already downloaded, which happens when two listed templates come from the same .zip.
        let packageSource = listedTemplate.packageUrl || listedTemplate.repo;
        if(!downloadedPackages[packageSource]) {
          let resolvedUrl;
          // If a template's downlaod type is url, we'll set the resolvedUrl to the template's packageUrl value.
          if(listedTemplate.downloadType === 'url'){

            if(!listedTemplate.packageUrl){
              throw new Error(`Invalid ADMX template entry! An ADMX template (${listedTemplate.displayName}) with a downloadType set to "url" has no packageUrl value. Please set a "packageUrl" value to the url where this ADMX template can be found and try running this script again.`);
            }

            resolvedUrl = listedTemplate.packageUrl;
          } else if(listedTemplate.downloadType === 'github release') {
            // Otherwise if the template is retreived from a GitHub release, we'll send a request to GitHub to get details of the latest release
            // and we'll download the zip

            if(!listedTemplate.repo) {
              throw new Error(`Invalid admx template entry! An ADMX template (${listedTemplate.displayName}) with a downloadType set to "github release" has no repo value. Please set a "repo" value to the GitHub repo where this ADMX template can be found and try running this script again.`);
            }

            let latestReleaseResponse = await sails.helpers.http.get.with({
              url: `https://api.github.com/repos/${listedTemplate.repo}/releases/latest`,
              headers: { 'User-Agent': 'fleetdm.com', 'Accept': 'application/vnd.github+json' },
            }).intercept((err)=>{
              return new Error(`When the regenerate-windows-admx-templates script sent an HTTP request to get the latest ADMX template for ${listedTemplate.displayName}, an error ocurred. Full error ${require('util').inspect(err)}`)
            });
            // Get the .zip files attached to the latest release.
            let zipAssets = _.filter(latestReleaseResponse.assets, (asset)=>{ return /\.zip$/i.test(asset.name); });
            let zipAsset;
            // If the template has a releaseItemContainsInName value, use the .zip whose name contains it.
            if(listedTemplate.releaseItemContainsInName) {
              zipAsset = _.find(zipAssets, (asset)=>{ return _.contains(asset.name, listedTemplate.releaseItemContainsInName); });
              if(!zipAsset) {
                throw new Error(`The latest ${listedTemplate.repo} release (${latestReleaseResponse.tag_name}) has no .zip whose name contains "${listedTemplate.releaseItemContainsInName}", so the ADMX template for ${listedTemplate.displayName} could not be downloaded.  The release's .zip files are: ${_.pluck(zipAssets, 'name').join(', ') || '(none)'}.  Update this template's "releaseItemContainsInName" value and try running this script again.`);
              }
            } else {
              // Otherwise, use the release's only .zip.
              if(zipAssets.length === 0) {
                throw new Error(`The latest ${listedTemplate.repo} release (${latestReleaseResponse.tag_name}) has no .zip, so the ADMX template for ${listedTemplate.displayName} could not be downloaded.`);
              }
              if(zipAssets.length > 1) {
                throw new Error(`The latest ${listedTemplate.repo} release (${latestReleaseResponse.tag_name}) has ${zipAssets.length} .zip files (${_.pluck(zipAssets, 'name').join(', ')}), so the script cannot tell which one has the ADMX template for ${listedTemplate.displayName}.  Set a "releaseItemContainsInName" value on this template to part of the right .zip's name and try running this script again.`);
              }
              zipAsset = zipAssets[0];
            }

            resolvedUrl = zipAsset.browser_download_url;

          } else {
            throw new Error(`Invalid ADMX template entry! An ADMX template (${listedTemplate.displayName}) has an unsupported "downloadType" value. Please set a "downloadType" value to either "github release" or "url" and try running this script again.`);
          }

          sails.log(`Downloading ${resolvedUrl}...`);
          // Use fetch to download the zip file containing the ADMX template.
          let response = await fetch(resolvedUrl);

          if(!response.ok) {
            throw new Error(`Downloading ${resolvedUrl} failed with status ${response.status}.`);
          }
          // Name the downloaded .zip by how many packages have been downloaded so far, so each one gets its own file in the temporary folder.
          let zipPath = path.join(workDir, `${Object.keys(downloadedPackages).length}.zip`);

          // Save the downloaded .zip to the temporary folder.
          fs.writeFileSync(zipPath, Buffer.from(await response.arrayBuffer()));

          // Google Update's package says nothing about its version, so the date it was published stands in.
          let lastModified = response.headers.get('last-modified');
          downloadedPackages[packageSource] = {
            zipPath,
            resolvedUrl,
            lastModified: lastModified ? new Date(lastModified).toISOString().slice(0, 10) : undefined,
            entries: childProcess.execFileSync('unzip', ['-Z1', zipPath]).toString().split('\n').filter(Boolean),
          };
        }
        // Get the downloaded package this template comes from.
        let downloaded = downloadedPackages[packageSource];
        // The folder inside the .zip that holds the app's .admx, where its .adml is also looked for.
        let admxFolder = path.posix.dirname(listedTemplate.admxPathInPackage);

        // Combine the listed template with its id, the display name in kebab case.
        let template = Object.assign({}, listedTemplate, {id: _.kebabCase(listedTemplate.displayName)});
        // Read the app's .admx out of the .zip.
        let admxText = readFromPackage(downloaded.zipPath, template.admxPathInPackage);

        // Throw an error if an ADMX template contains "]]>", which would break how it will be injected into ADMX-backed XML policies
        if(_.contains(admxText, ']]>')) {
          throw new Error(`${template.admxPathInPackage} contains "]]>", so it cannot be embedded in a CDATA section.`);
        }

        // Name the template after the namespace its .admx defines, without "Policies", e.g., Google.Policies.Chrome is GoogleChrome.  Windows builds every policy's LocURI from this name, and a vendor's namespace is meant to be unique, so two templates never share one.
        let targetNamespace = parseAttributes((admxText.match(/<target\b((?:[^>"]|"[^"]*")*?)\/?>/) || [])[1] || '').namespace;
        if(!targetNamespace) {
          throw new Error(`${template.admxPathInPackage} has no <target namespace="..."/> element, so the script cannot name the template.  Check that admxPathInPackage points to an .admx file.`);
        }
        let appName = _.map(_.reject(targetNamespace.split('.'), (segment)=>{ return /^policies$/i.test(segment); }), _.capitalize).join('').replace(/[^A-Za-z0-9]/g, '');
        let templateWithTheSameAppName = _.find(templatesToWrite, (written)=>{ return written.entry.appName === appName; });
        if(templateWithTheSameAppName) {
          throw new Error(`${template.displayName} and ${templateWithTheSameAppName.template.displayName} both define the ${targetNamespace} namespace, so they would install over each other.  Remove one of them from TEMPLATES.`);
        }

        // Use Chrome's version comment as the template's version, or the date the vendor's package was published when there is none.
        let vendorVersion = (admxText.match(/<!--\s*chrome version:\s*([\d.]+)\s*-->/) || [])[1] || downloaded.lastModified;
        // Save the template under its file name inside the .zip (chrome.admx, firefox.admx).
        let savedAs = path.posix.basename(template.admxPathInPackage);
        templatesToWrite.push({
          template,
          admxText,
          savedAs,
          entry: {
            id: template.id,
            displayName: template.displayName,
            keywords: template.keywords,
            vendorVersion,
            sourceUrl: downloaded.resolvedUrl,
            appName,
            path: `admx-templates/${template.id}/${savedAs}`,
            bytes: Buffer.byteLength(admxText, 'utf8'),
            installLocUri: `./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/${appName}/Policy/${appName}AdmxFile`,
          },
        });

        // Find the template's en-US .adml, which holds the display names, descriptions and element labels its .admx refers to.
        let admlPathInPackage = `${admxFolder}/en-US/${path.posix.basename(template.admxPathInPackage).replace(/\.admx$/i, '.adml')}`;
        if(!_.contains(downloaded.entries, admlPathInPackage)) {
          throw new Error(`Could not find the en-US .adml for ${template.displayName}.  The script expects it at ${admlPathInPackage} in ${downloaded.resolvedUrl}, beside the .admx in an en-US folder.  Check that admxPathInPackage is correct, and that the vendor's package includes an en-US .adml.`);
        }
        // Read the .adml's contents.
        let admlText = readFromPackage(downloaded.zipPath, admlPathInPackage);
        let strings = {};
        // Match each string in the .adml, e.g., <string id="HomepageLocation">Configure the home page URL</string>.
        let stringRegExp = /<string\s+id="([^"]+)"\s*>([\s\S]*?)<\/string>/g;
        let match;
        // Loop through the matched strings, and store each one's decoded text by its id.
        while ((match = stringRegExp.exec(admlText)) !== null) {
          strings[match[1]] = decodeEntities(match[2]).trim();
        }
        let labelsByPresentationId = {};
        // Match each presentation in the .adml, e.g., <presentation id="HomepageLocation"><textBox refId="HomepageLocation"><label>Home page URL</label></textBox></presentation>.
        let presentationRegExp = /<presentation\s+id="([^"]+)"\s*>([\s\S]*?)<\/presentation>/g;
        // Loop through the matched presentations, and store the label of each element it lays out (the text beside its box in the Group Policy editor) by the element's id.
        while ((match = presentationRegExp.exec(admlText)) !== null) {
          let labels = {};
          let controlRegExp = /<(\w+)\b((?:[^>"]|"[^"]*")*?)(?:\/>|>([\s\S]*?)<\/\1>)/g;
          let control;
          while ((control = controlRegExp.exec(match[2])) !== null) {
            let refId = (control[2].match(/\brefId="([^"]+)"/) || [])[1];
            if(!refId) {
              continue;
            }
            let inner = control[3] || '';
            // Use the text of the control's <label>, or the control's own text when it has none, e.g., <checkBox refId="HomepageLocked">Don't allow the homepage to be changed.</checkBox>.
            let label = (inner.match(/<label>([\s\S]*?)<\/label>/) || [])[1] || inner.replace(/<[^>]+>[\s\S]*?<\/[^>]+>/g, '').replace(/<[^>]+>/g, '');
            labels[refId] = decodeEntities(label).replace(/\s+/g, ' ').trim();
          }
          labelsByPresentationId[match[1]] = labels;
        }

        let unresolvedStrings = [];
        // Look up the text a $(string.Id) reference in the .admx points to, and record any reference the .adml has no string for.
        let resolve = (reference)=>{
          // Get the string id from a reference like $(string.HomepageLocation), or undefined when the value is plain text rather than a reference.
          let stringId = (String(reference || '').match(/^\$\(string\.([^)]+)\)$/) || [])[1];

          if(!stringId) {
            return reference;
          }

          if(strings[stringId] === undefined) {
            unresolvedStrings.push(stringId);
            return undefined;
          }
          return strings[stringId];
        };

        let categoriesByName = {};
        // Match each category in the .admx, e.g., <category name="Startup" displayName="$(string.Startup_group)"><parentCategory ref="googlechrome"/></category>.
        let categoryRegExp = /<category\b((?:[^>"]|"[^"]*")*?)(?:\/>|>([\s\S]*?)<\/category>)/g;
        // Loop through the matched categories, and store each one's parent by its name, so a policy's full category path can be built below.
        while ((match = categoryRegExp.exec(admxText)) !== null) {
          let attributes = parseAttributes(match[1]);
          categoriesByName[attributes.name] = {
            name: attributes.name,
            parentRef: (String(match[2] || '').match(/<parentCategory\s+ref="([^"]+)"/) || [])[1],
          };
        }

        let nodesForThisTemplate = [];
        let droppedForRegistryKey = [];
        // Match each policy in the .admx, e.g., <policy name="HomepageLocation" class="Both" key="Software\Policies\Google\Chrome">...</policy>.
        let policyRegExp = /<policy\b((?:[^>"]|"[^"]*")*?)(?:\/>|>([\s\S]*?)<\/policy>)/g;
        // Loop through the matched policies, and build a node for each one the generator can offer.
        while ((match = policyRegExp.exec(admxText)) !== null) {

          // Read the policy's attributes (name, class, key, displayName, explainText, presentation).
          let attributes = parseAttributes(match[1]);

          // The XML inside the policy element: its parent category, and the elements it takes values for.
          let body = match[2] || '';

          // Windows builds the area from the category chain joined by ~.  A parent in another namespace
          // (prefix:Name) ends the chain, which is what puts Chrome's policies under
          // Chrome~Policy~googlechrome~... rather than under Google's Cat_Google.
          let categoryPath = [];
          // Start from the category the policy's <parentCategory> points to.
          let category = categoriesByName[(body.match(/<parentCategory\s+ref="([^"]+)"/) || [])[1]];
          // Walk up through each category's parent, adding it to the front of the path, until reaching the root or a parent in another template's namespace.
          while (category) {
            categoryPath.unshift(category.name);
            if(!category.parentRef || _.contains(category.parentRef, ':') || _.contains(categoryPath, category.parentRef)) {
              break;
            }
            category = categoriesByName[category.parentRef];
          }
          if(categoryPath.length === 0) {
            continue;
          }
          if(template.onlyCategories && !_.contains(template.onlyCategories, _.last(categoryPath))) {
            continue;
          }
          // Chrome keeps every policy it has retired in the template, under RemovedPolicies, so that existing
          // Group Policy objects still open.  Setting one does nothing, so it is never worth offering.
          if(_.contains(categoryPath, 'RemovedPolicies')) {
            continue;
          }

          // Find the presentation that labels this policy's elements in the Group Policy editor.
          let presentationId = (String(attributes.presentation || '').match(/^\$\(presentation\.([^)]+)\)$/) || [])[1];
          let labels = labelsByPresentationId[presentationId] || {};
          let registryKeys = _.compact([attributes.key]);
          let admxElements = [];
          // The XML inside the policy's <elements>, which defines each value the policy takes when enabled.
          let elementsBody = (body.match(/<elements>([\s\S]*?)<\/elements>/) || [])[1] || '';
          // Match each element the policy takes a value for, e.g., <text id="HomepageLocation" valueName="HomepageLocation"/>.
          let elementRegExp = /<(text|decimal|longDecimal|boolean|enum|list|multiText)\b((?:[^>"]|"[^"]*")*?)(?:\/>|>([\s\S]*?)<\/\1>)/g;
          let elementMatch;
          // Loop through the matched elements, and record each one's type, id, label and allowed values.
          while ((elementMatch = elementRegExp.exec(elementsBody)) !== null) {
            let elementAttributes = parseAttributes(elementMatch[2]);
            if(elementAttributes.key) {
              registryKeys.push(elementAttributes.key);
            }
            let element = {
              type: elementMatch[1],
              id: elementAttributes.id,
              label: labels[elementAttributes.id] || undefined,
              required: elementAttributes.required === 'true' || undefined,
            };
            // Record the allowed range of a number element.
            if(element.type === 'decimal' || element.type === 'longDecimal') {
              element.min = elementAttributes.minValue !== undefined ? Number(elementAttributes.minValue) : undefined;
              element.max = elementAttributes.maxValue !== undefined ? Number(elementAttributes.maxValue) : undefined;
            }
            if(element.type === 'list') {
              // Without explicitValue the admin supplies only values and Windows names them (valuePrefix + 1,
              // 2, ...), but the <data> payload still carries name/value pairs either way.
              element.explicitValue = elementAttributes.explicitValue === 'true' || undefined;
            }
            // Record the choices of an enum (drop-down) element: each item's value and label.
            if(element.type === 'enum') {
              element.items = [];
              let itemRegExp = /<item\b((?:[^>"]|"[^"]*")*?)>([\s\S]*?)<\/item>/g;
              let itemMatch;
              // Loop through the enum's <item> elements.
              while ((itemMatch = itemRegExp.exec(elementMatch[3] || '')) !== null) {
                // The XML inside the item's <value>, e.g., <decimal value="5"/>.
                let valueBody = (itemMatch[2].match(/<value>([\s\S]*?)<\/value>/) || [])[1] || '';
                // Use the item's number value when it has one.
                let value = (valueBody.match(/<(?:decimal|longDecimal)\s+value="([^"]*)"/) || [])[1];

                // Otherwise use the item's string value, e.g., <string>always</string>.
                if(value === undefined) {
                  value = decodeEntities((valueBody.match(/<string>([\s\S]*?)<\/string>/) || [])[1] || '');
                }

                element.items.push({value, label: resolve(parseAttributes(itemMatch[1]).displayName)});
              }
            }
            admxElements.push(_.omit(element, _.isUndefined));
          }
          // Find the first registry key this policy writes to that Windows won't let an installed template set.
          let blockedKey = _.find(registryKeys, (registryKey)=>{
            let normalized = String(registryKey).toLowerCase().replace(/\\?$/, '\\');
            return _.any(BLOCKED_REGISTRY_PREFIXES, (prefix)=>{ return _.startsWith(normalized, prefix); }) &&
              !_.any(ALLOWED_REGISTRY_PREFIXES, (prefix)=>{ return _.startsWith(normalized, prefix.replace(/\\?$/, '\\')); });
          });
          // Leave out a policy that writes to a blocked key, and record it for the warning logged below.
          if(blockedKey) {
            droppedForRegistryKey.push(`${attributes.name} (${blockedKey})`);
            continue;
          }

          // Keep only the first sentence of the policy's explanation: the full text is documentation, and in the
          // lookup index long text pulls the model towards whichever policy shares a word with the request.
          let explanation = String(resolve(attributes.explainText) || '').replace(/\s+/g, ' ').trim().split(/(?<=[a-z0-9)]\.)\s+(?=[A-Z])/)[0];
          if(explanation.length > 200) {
            explanation = explanation.slice(0, 200).replace(/\s+\S*$/, '') + '…';
          }
          // Turn the policy's ADMX class into the scopes it can be set in: User, Device, or both.
          let scopes = attributes.class === 'User' ? ['User'] : (attributes.class === 'Machine' ? ['Device'] : ['Device', 'User']);
          // Build the policy's area the way Windows names it after the template is installed: <AppName>~Policy~<category path>.
          let area = `${appName}~Policy~${categoryPath.join('~')}`;
          nodesForThisTemplate.push({
            csp: 'Policy',
            admxTemplate: template.id,
            area,
            name: attributes.name,
            // The policy's LocURI, under ./Device/ unless the policy can only be set for a user.
            locUri: `./${scopes[0]}/Vendor/MSFT/Policy/Config/${area}/${attributes.name}`,
            // The policy's display name and first sentence of explanation, since several templates reuse one explanation across many policies.
            description: _.compact([resolve(attributes.displayName), explanation]).join(' -- '),
            scopes,
            format: 'chr',
            accessType: 'Add, Delete, Get, Replace',
            admxElements,
            deprecated: _.contains(categoryPath, 'DeprecatedPolicies') || undefined,
          });
        }

        if(droppedForRegistryKey.length > 0) {
          sails.log.warn(`${template.id}: dropped ${droppedForRegistryKey.length} policies that write to registry keys Windows will not let an ingested template set: ${droppedForRegistryKey.join(', ')}`);
        }
        if(unresolvedStrings.length > 0) {
          sails.log.warn(`${template.id}: ${_.uniq(unresolvedStrings).length} $(string.*) references have no en-US string (first few: ${_.uniq(unresolvedStrings).slice(0, 5).join(', ')}).`);
        }
        if(nodesForThisTemplate.length === 0) {
          throw new Error(`Refusing to write: ${template.id} produced no policies.  The vendor's file has probably changed shape.`);
        }
        sails.log(`${template.id} ${vendorVersion}: ${nodesForThisTemplate.length} policies in ${_.uniq(_.pluck(nodesForThisTemplate, 'area')).length} areas (${Math.round(Buffer.byteLength(admxText, 'utf8') / 1024)} KB).`);
        allNodes = allNodes.concat(nodesForThisTemplate);
      }
    } finally {
      fs.rmSync(workDir, {recursive: true, force: true});
    }

    let outputPath = path.join(SCHEMA_DIR, 'windows-admx-templates.json');
    if(dry) {
      sails.log(`Dry run -- not writing.  Would have written ${templatesToWrite.length} templates and ${allNodes.length} policies to ${outputPath}.`);
      return;
    }

    // Delete the saved templates before writing them again, so a template removed from TEMPLATES does not leave its file behind.
    fs.rmSync(path.join(SCHEMA_DIR, 'admx-templates'), {recursive: true, force: true});
    for (let {template, admxText, savedAs} of templatesToWrite) {
      let templateDir = path.join(SCHEMA_DIR, 'admx-templates', template.id);
      fs.mkdirSync(templateDir, {recursive: true});
      fs.writeFileSync(path.join(templateDir, savedAs), admxText);
    }

    await sails.helpers.fs.writeJson.with({
      destination: outputPath,
      json: {
        generatedAt: (new Date()).toISOString(),
        templates: _.pluck(templatesToWrite, 'entry'),
        // Sorted so a regeneration produces a readable diff.
        nodes: _.sortBy(allNodes, (node)=>{ return `${node.area}/${node.name}`; }),
      },
      force: true
    });

    sails.log(`\nWrote ${templatesToWrite.length} templates to ${path.join(SCHEMA_DIR, 'admx-templates')} and ${allNodes.length} policies to ${outputPath}.`);

  }


};


// Reads an XML start tag's attributes into a dictionary.
function parseAttributes(attributeText) {
  let attributes = {};
  // Match each name="value" pair in the start tag, e.g., name="HomepageLocation".
  let attributeRegExp = /([\w:]+)\s*=\s*"([^"]*)"/g;
  let match;
  while ((match = attributeRegExp.exec(attributeText)) !== null) {
    attributes[match[1]] = decodeEntities(match[2]);
  }
  return attributes;
}

// Replaces XML entities (&amp;, &lt;, &#xF000;, ...) with the characters they stand for.
function decodeEntities(str) {
  return String(str)
    .replace(/&#(\d+);/g, (unusedMatch, code)=>{ return String.fromCharCode(Number(code)); })
    .replace(/&#x([0-9a-fA-F]+);/g, (unusedMatch, code)=>{ return String.fromCharCode(parseInt(code, 16)); })
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, '\'')
    .replace(/&amp;/g, '&');
}
