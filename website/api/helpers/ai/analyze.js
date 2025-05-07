module.exports = {


  friendlyName: 'Analyze',


  description: 'Score the provided data along multiple custom dimensions.',
  // TODO: Consider renaming this helper to `.weigh()` to be more specific


  extendedDescription: 'e.g. build an index for a "recommended product" feature in an ecommerce site by scoring a product along custom dimensions',


  cacheable: true,


  inputs: {

    data: {
      type: 'json',
      required: true
    },

    dimensions: {
      type: [ 'string' ],
      required: true,
      example: [
        'night on the town',
        'formal',
        'polyester fabric',
        'wool fabric'
      ]
    },

    // FUTURE: Decide whether to introduce an option (or maybe just a dimension naming convention) that lets you indicate that a particular dimension should be weighed as a binary "yes vs no" decision (i.e. 0 or 1)

    // TODO: Decide whether to include percentage option where weights all add up to 100% (e.g. 0.35, 0.4, 0.15)
    // ....should it be "how true" / "how relevant" is each dimension, on a consistent scale?
    // probably.  And then an optional flag you pass in if you want it to add up to 100%.
  },


  exits: {

    success: {
      outputFriendlyName: 'Weights',
      outputDescription: 'The weights/scores of this data along each dimension, expressed as a number from 0 to 1.',
      outputType: {},
      outputExample: {
        'night on the town': 0.3,
        'formal': 0.1,
        'polyester fabric': 1,
        'wool fabric': 0,
        'cotton fabric': 1,
      },

    },

  },


  fn: async function ({}) {
    // TODO

    // TODO: Limit (round) the precision of decimal places for better userland experience.
  }


};

