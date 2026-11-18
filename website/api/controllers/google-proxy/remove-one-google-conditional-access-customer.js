module.exports = {


  friendlyName: 'Remove one Google conditional access customer',


  description: 'Deletes the Google conditional access registration for a provided Google customer ID.',


  inputs: {
    googleCustomerId: {
      type: 'string',
      required: true,
    },
    fleetServerSecret: {
      type: 'string',
      required: true,
    },
  },


  exits: {
    success: {
      description: 'The Google conditional access registration has been deleted.'
    },
    customerNotFound: {
      description: 'A Google conditional access registration could not be found using the provided information.',
      responseType: 'notFound',
    },
  },


  fn: async function ({googleCustomerId, fleetServerSecret}) {

    // Google's device APIs use the customer ID without the leading "C", so accept either form and match it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    let destroyResult = await GoogleConditionalAccessServer.destroyOne({googleCustomerId, fleetServerSecret});
    if(!destroyResult) {
      throw 'customerNotFound';
    }

    // All done.
    return;

  }


};
