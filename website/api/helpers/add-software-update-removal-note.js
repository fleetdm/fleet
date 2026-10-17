module.exports = {


  friendlyName: 'Add software update removal note',


  description: 'Add a delivery note to a generated .mobileconfig that uses the software update settings Apple removed in macOS 27.',


  extendedDescription:
`The prompt asks for this note, but the delivery-notes rules favour an empty string, and the model leaves it out in
a few runs.  The profile still installs on older macOS, so the admin needs to hear it from the notes.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated .mobileconfig.'
    },

    deliveryNotes: {
      type: 'string',
      defaultsTo: '',
      description: 'The delivery notes the model wrote.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Delivery notes',
      outputType: 'string',
    },

  },


  fn: async function ({profile, deliveryNotes}) {

    let usesRemovedSettings = /<string>com\.apple\.SoftwareUpdate<\/string>|<key>(forceDelayed\w*SoftwareUpdates|enforcedSoftwareUpdate\w*Delay)<\/key>/.test(profile);
    if(!usesRemovedSettings || /macOS 27/i.test(deliveryNotes)) {
      return deliveryNotes;
    }
    return _.compact([deliveryNotes, 'These software update settings stop working in macOS 27, where the software update settings declaration replaces them.']).join(' ');

  }


};
