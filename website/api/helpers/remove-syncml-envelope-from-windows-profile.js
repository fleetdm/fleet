module.exports = {


  friendlyName: 'Remove SyncML envelope from Windows profile',


  description: 'Strip an XML declaration and SyncML envelope from a generated Windows CSP profile, leaving its commands.',


  extendedDescription:
`The prompt forbids both, and the model still wraps the commands in <?xml ...?><SyncML><SyncBody> in a few runs
out of a hundred.  MDMs reject that on upload, since the envelope carries session state only the MDM can fill in.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated Windows CSP profile.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Profile without SyncML envelope',
      outputType: 'string',
    },

  },


  fn: async function ({profile}) {

    // Only the outermost layers are touched, so a declaration or SyncML inside a CDATA value is left alone.
    let unwrapped = profile.trim().replace(/^<\?xml[^>]*\?>\s*/, '');
    let syncML = unwrapped.match(/^<SyncML\b[^>]*>([\s\S]*)<\/SyncML>$/);
    if(syncML) {
      unwrapped = syncML[1].trim().replace(/^<SyncHdr\b[\s\S]*?<\/SyncHdr>\s*/, '');
      let syncBody = unwrapped.match(/^<SyncBody\b[^>]*>([\s\S]*)<\/SyncBody>$/);
      if(syncBody) {
        unwrapped = syncBody[1].trim();
      }
      unwrapped = unwrapped.replace(/\s*<Final\s*\/>$/, '');
    }
    return unwrapped === profile.trim() ? profile : unwrapped;

  }


};
