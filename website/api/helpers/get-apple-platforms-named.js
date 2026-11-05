module.exports = {


  friendlyName: 'Get Apple platforms named',


  description: 'Find which Apple platforms an IT admin\'s instructions name, in the names Apple\'s schemas use for them.',


  sync: true,


  inputs: {

    naturalLanguageInstructions: {
      type: 'string',
      required: true,
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Platforms named',
      outputDescription: 'e.g. ["iOS", "macOS"], or an empty array when the instructions name no platform.',
    },

  },


  fn: function ({naturalLanguageInstructions}) {

    // iPadOS has no platform of its own in Apple's schemas: its availability is iOS's.  "Mac" is matched
    // case-sensitively so that a MAC address is not a Mac.
    const PATTERNS_BY_PLATFORM = {
      macOS: [/\bmac ?os\b/i, /\bos x\b/i, /\b(?:macbook|imac|mac ?mini|mac ?studio|mac ?pro)s?\b/i, /\b[Mm]acs?\b/],
      iOS: [/\bi(?:pad)?os\b/i, /\biphones?\b/i, /\bipads?\b/i],
      tvOS: [/\btv ?os\b/i, /\bapple ?tvs?\b/i],
      visionOS: [/\bvision ?os\b/i, /\bvision pro\b/i],
      watchOS: [/\bwatch ?os\b/i, /\bapple watch(?:es)?\b/i],
    };
    return _.filter(Object.keys(PATTERNS_BY_PLATFORM), (platformName)=>{
      return _.any(PATTERNS_BY_PLATFORM[platformName], (pattern)=>{ return pattern.test(naturalLanguageInstructions); });
    });

  }


};
