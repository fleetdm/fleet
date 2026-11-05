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

    // The Fleet server stores the customer ID without the leading "C" (#56649), and fleetdm.com stores it with one.
    googleCustomerId = 'C' + googleCustomerId.replace(/^C/, '');

    let informationAboutThisCustomer = await GoogleConditionalAccessServer.findOne({googleCustomerId, fleetServerSecret});
    if(!informationAboutThisCustomer) {
      throw 'customerNotFound';
    }

    await GoogleConditionalAccessServer.destroyOne({id: informationAboutThisCustomer.id});

    // All done.
    return;

  }


};
