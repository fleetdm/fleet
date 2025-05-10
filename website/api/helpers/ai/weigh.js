module.exports = {


  friendlyName: 'Weigh',


  description: 'Score the provided data along multiple custom dimensions.',


  extendedDescription: 'e.g. build an index for a "recommended product" feature in an ecommerce site by scoring a product for future searching/querying on product detail pages in the "Recommended for you" section.',


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


  fn: async function ({ data, dimensions }) {

    // TODO: Limit (round) the precision of decimal places for better userland experience.

    let prompt = 'Given some data and a set of dimensions, score the data on a scale from 0 to 1 along each dimension, using a decimal precision of no more than one decimal place.';

    prompt += 'Data: ```\n';
    prompt += `${JSON.stringify(data)}\n`;
    prompt += '```\n';
    prompt += '\n';
    prompt += 'Dimensions:\n';
    for (let dimension of dimensions) {
      prompt += ` • ${dimension}\n`;
    }//∞
    prompt += '\n';
    prompt += 'Respond only with JSON in this data shape: `{"foo": 0.8, "bar": 0.4 }`';

    let weights = await sails.helpers.flow.build(async ()=>{
      let parsedPromptResponse = await sails.helpers.ai.prompt.with({
        expectJson: true,
        baseModel: 'o4-mini-2025-04-16',
        prompt: prompt,
      })
      .retry('jsonExpectationFailed');

      if (!_.isObject(parsedPromptResponse) || _.isArray(parsedPromptResponse) || _.intersection(dimensions,Object.keys(parsedPromptResponse)).length === dimensions.length) {
        throw new Error('Response from LLM does not match the expected format for weights derived from the provided dimensions.  The LLM said: \n```\n'+parsedPromptResponse+'\n```\n\nBut the provided dimensions were: \n```\n'+require('util').inspect(dimensions, {depth: null})+'\n```');
      }

      return parsedPromptResponse;

    }).retry();

    return weights;
  }


};

