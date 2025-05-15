module.exports = {


  friendlyName: 'Satisfy',


  description: 'Modify some data such that it satisfies one or more constraints.',


  extendedDescription: 'e.g. wedding seating chart, generate work schedule',


  sideEffects: 'cacheable',


  inputs: {
    data: {
      type: 'json',
      required: true
    },

    constraints: {
      description: 'A list of constraints to impose upon the provided data and any changes to it.',
      type: [ 'string' ],
      required: true,
      example: [ `Every table must have no more than 2 empty seats.`, `Couples with the same last name should sit together at the same table.` ]
    },

    changes: {
      description: 'An optional list of changes to make to the data, in order, keeping with the constraints all the while.',
      type: [ 'string' ],
      example: [`First take out the Brimsly's and Jestine Friggledour`, `Then replace the Smith's with Ferngalia, if you can`]
    },
  },


  exits: {

    success: {
      outputType: 'json',
      outputDescription: 'The modified data.',
      extendedDescription: 'Note that this is a deep clone returned from the LLM.  (The original data is not modified in-place.)'
    },

  },


  fn: async function ({data, constraints, changes}) {

    let prompt = `Given some data and a set of constraints, make sure the data matches all of those constraints.`;

    prompt += 'Data: ```\n';
    prompt += `${JSON.stringify(data)}\n`;
    prompt += '```\n';
    prompt += '\n';
    prompt += 'Constraints:\n';
    for (let constraint of constraints) {
      prompt += ` • ${constraint}\n`;
    }//∞
    prompt += '\n';
    if (changes) {
      prompt += 'And also apply all of these changes, adhering to all constraints:\n';
      for (let idx=0; idx<changes.length; idx++) {
        prompt += ` ${idx}. ${changes[idx]}\n`;
      }//∞
      prompt += '\n';
    }//ﬁ

    return await sails.helpers.ai.prompt.with({
      expectJson: true,
      baseModel: 'o4-mini-2025-04-16',
      prompt: prompt,
    })
    .retry('jsonExpectationFailed');
  }


};

