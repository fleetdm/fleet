module.exports = {


  friendlyName: 'Add ADMX install commands to Windows profile',


  description: 'Add the ADMXInstall command for every third-party ADMX template a generated Windows CSP profile uses.',


  extendedDescription:
`A third-party app's policy (GoogleChrome~Policy~googlechrome~Startup/HomepageLocation) only exists on a device
once the app's ADMX template has been ingested, so the profile has to carry the template too.  The model
cannot write it -- chrome.admx alone is ~450KB -- so the generator tells it not to, and this adds it
from the copy vendored in profile-generator/schema/admx-templates/, ahead of the policies that need it.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated Windows CSP profile: a sequence of <Add>, <Replace>, <Exec> or <Atomic> commands.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Profile with ADMX install commands',
      outputDescription: '{profile, admxInstallProfiles?: [{filename, profile}], admxTemplatesInstalled: [{id, displayName, vendorVersion}], unknownPolicies: [locUri], unknownDataIds: ["Policy: id"], deliveryNotes: [sentence]}',
      outputType: 'ref',
    },

  },


  fn: async function ({profile}) {

    let path = require('path');
    let fs = require('fs');

    // Fleet's limit on a profile's content.  Its MaxProfileSize is 1.5MiB, but the extra half is headroom for
    // base64 encoding, and the upload error reports 1 MB.
    const MAX_PROFILE_BYTES = 1000 * 1000;

    // Read the ADMX template reference, and throw an error that says how to build it if it is missing.
    let admxTemplateFilePath = path.resolve(sails.config.appPath, 'profile-generator/schema/windows-admx-templates.json');
    let admxTemplateFile;
    try {
      admxTemplateFile = require(admxTemplateFilePath);
    } catch (err) {
      throw new Error(
        `Could not read the Windows ADMX template reference at ${admxTemplateFilePath}.  Run ` +
        `\`sails run regenerate-windows-admx-templates\` to build it.  Full error: ${err.message}`
      );
    }
    let templatesById = _.indexBy(admxTemplateFile.templates, 'id');
    let templateIdsByAppName = {};
    for (let template of admxTemplateFile.templates) {
      templateIdsByAppName[template.appName.toLowerCase()] = template.id;
    }
    let nodesByLowercaseLocUri = {};
    for (let node of admxTemplateFile.nodes) {
      for (let scope of node.scopes) {
        nodesByLowercaseLocUri[`./${scope}/Vendor/MSFT/Policy/Config/${node.area}/${node.name}`.toLowerCase()] = node;
      }
    }

    // Anything the model wrote at an ADMXInstall path is dropped. It is told not to, but still sometimes tries, and what it produces is a truncated or invented template.
    let commandRegExp = /[ \t]*<(Add|Replace|Exec)\b[^>]*>[\s\S]*?<\/\1>[ \t]*\n?/g;
    let profileWithoutInstalls = profile.replace(commandRegExp, (command)=>{
      return /ConfigOperations\/ADMXInstall/i.test(command) ? '' : command;
    });

    let templateIdsUsed = [];
    let unknownPolicies = [];
    let unknownDataIds = [];
    // Each command is read item by item, since one <Add> or <Replace> can carry several <Item>s.  A command
    // whose LocURI sits outside any <Item> -- the model sometimes puts <Target> beside its <Item> -- is read
    // whole, since a policy missed here is a profile shipped without the template it needs.
    let commandMatch;
    let commandsRegExp = new RegExp(commandRegExp.source, 'g');
    // Loop through the profile's commands, and record which templates its third-party policies need.
    while ((commandMatch = commandsRegExp.exec(profileWithoutInstalls)) !== null) {
      let command = commandMatch[0];
      let itemsWithALocUri = _.filter(command.match(/<Item>[\s\S]*?<\/Item>/g) || [], (item)=>{ return /<LocURI>/.test(item); });
      for (let segment of itemsWithALocUri.length > 0 ? itemsWithALocUri : [command]) {
        let locUri = ((segment.match(/<LocURI>\s*([\s\S]*?)\s*<\/LocURI>/) || [])[1] || '').trim();
        let areaMatch = locUri.match(/^\.\/(?:Device|User)\/Vendor\/MSFT\/Policy\/Config\/([^/~]+)~Policy~/i);
        if(!areaMatch) {
          continue;
        }
        let node = nodesByLowercaseLocUri[locUri.toLowerCase()];
        if(!node) {
          unknownPolicies.push(locUri);
          // Still installed when the app is recognized: a policy name that is slightly off fails on its own,
          // while a missing template takes every other policy in the profile down with it.
          if(templateIdsByAppName[areaMatch[1].toLowerCase()]) {
            templateIdsUsed.push(templateIdsByAppName[areaMatch[1].toLowerCase()]);
          }
          continue;
        }
        templateIdsUsed.push(node.admxTemplate);
        let knownIds = _.pluck(node.admxElements, 'id');
        let dataIdRegExp = /<data\s+id="([^"]+)"/gi;
        let dataIdMatch;
        // Loop through the policy's <data> elements, and record any id its template does not define.
        while ((dataIdMatch = dataIdRegExp.exec(segment)) !== null) {
          if(!_.contains(knownIds, dataIdMatch[1])) {
            unknownDataIds.push(`${node.name}: ${dataIdMatch[1]}`);
          }
        }
      }
    }

    let templateIdsToInstall = _.uniq(templateIdsUsed);
    // Get the id, display name and version of each template being installed, for the caller and the delivery notes.
    let admxTemplatesInstalled = _.map(templateIdsToInstall, (templateId)=>{
      return _.pick(templatesById[templateId], ['id', 'displayName', 'vendorVersion']);
    });

    // What the admin needs to know that the profile does not show, for the caller to add to its delivery notes.
    let deliveryNotesFor = (admxInstallProfiles)=>{
      let notes = [];
      let templateNames = _.map(admxTemplatesInstalled, (installed)=>{ return `${installed.displayName} (${installed.vendorVersion})`; }).join(', ');
      if(admxInstallProfiles) {
        notes.push(`The ADMX templates these policies need -- ${templateNames} -- are too large to share a file with them, so each is in its own profile (${_.pluck(admxInstallProfiles, 'filename').join(', ')}); upload those too, and expect these policies to fail on a host until its template profile has been delivered.`);
      } else if(admxTemplatesInstalled.length > 0) {
        notes.push(`The ADMX template${admxTemplatesInstalled.length > 1 ? 's' : ''} these policies need -- ${templateNames} -- ${admxTemplatesInstalled.length > 1 ? 'are' : 'is'} installed at the top of this profile; if another profile installs a different version, whichever is delivered last wins.`);
      }
      // Name any third-party policy that is not in its template, since the device will reject it.
      if(unknownPolicies.length > 0) {
        notes.push(`Check ${unknownPolicies.join(', ')}: ${unknownPolicies.length > 1 ? 'they are' : 'it is'} not in the vendor's ADMX template, so the device will reject ${unknownPolicies.length > 1 ? 'them' : 'it'}.`);
      }
      // Name any <data> id that is not in its policy's template.
      if(unknownDataIds.length > 0) {
        notes.push(`Check the <data> id${unknownDataIds.length > 1 ? 's' : ''} ${unknownDataIds.join(', ')}: not in the vendor's ADMX template.`);
      }
      return notes;
    };

    if(templateIdsToInstall.length === 0) {
      return { profile: profileWithoutInstalls, admxTemplatesInstalled, unknownPolicies, unknownDataIds, deliveryNotes: deliveryNotesFor() };
    }

    // Shaped exactly like the install command in Fleet's tested ADMX profiles (docs/solutions/windows/configuration-profiles/):
    // a <Replace>, and the template's <?xml ...?> declaration directly after <![CDATA[, since a line break between
    // the two made ADMXInstall fail with status 500.
    let installCommandFor = (templateId)=>{
      let template = templatesById[templateId];
      let admxText = fs.readFileSync(path.resolve(sails.config.appPath, 'profile-generator/schema', template.path), 'utf8');
      return [
        '<Replace>',
        '  <Item>',
        '    <Meta>',
        '      <Format xmlns="syncml:metinf">chr</Format>',
        '    </Meta>',
        '    <Target>',
        `      <LocURI>${template.installLocUri}</LocURI>`,
        '    </Target>',
        `    <Data><![CDATA[${admxText.replace(/^\s+/, '')}]]></Data>`,
        '  </Item>',
        '</Replace>',
      ].join('\n');
    };

    // Fleet requires an <Atomic> to be the profile's only top-level command, so in a profile wrapped in one
    // the installs go inside it, ahead of the policies.
    let installCommands = _.map(templateIdsToInstall, installCommandFor).join('\n');
    let combinedProfile;
    let atomicOpening = profileWithoutInstalls.match(/^\s*<Atomic\b[^>]*>\s*/);
    if(atomicOpening) {
      combinedProfile = atomicOpening[0] + installCommands + '\n' + profileWithoutInstalls.slice(atomicOpening[0].length);
    } else {
      combinedProfile = installCommands + '\n' + profileWithoutInstalls.replace(/^\s+/, '');
    }

    // If the profile with its install commands fits within Fleet's size limit, return it as a single profile.
    if(Buffer.byteLength(combinedProfile, 'utf8') <= MAX_PROFILE_BYTES) {
      return { profile: combinedProfile, admxTemplatesInstalled, unknownPolicies, unknownDataIds, deliveryNotes: deliveryNotesFor() };
    } else {
      // Otherwise, return the policies without the install commands, and each template's install command as a profile of its own.
      let admxInstallProfiles = _.map(templateIdsToInstall, (templateId)=>{
        return {
          filename: `install-${templateId}-admx-template.xml`,
          profile: installCommandFor(templateId) + '\n',
        };
      });
      return { profile: profileWithoutInstalls, admxInstallProfiles, admxTemplatesInstalled, unknownPolicies, unknownDataIds, deliveryNotes: deliveryNotesFor(admxInstallProfiles) };
    }

  }


};
