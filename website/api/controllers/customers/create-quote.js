module.exports = {


  friendlyName: 'Create quote',


  description: '',


  inputs: {

    numberOfHosts: {
      type: 'number',
      required: true,
    },

  },


  exits: {

  },


  fn: async function ({ numberOfHosts }) {

    // Determine the price, 1 dollar * host * month (Billed anually)
    let price = 2.00 * numberOfHosts * 12;

    let quote = await Quote.create({
      numberOfHosts: numberOfHosts,
      quotedPrice: price,
      organization: this.req.me.organization,
      user: this.req.me.id,
    }).fetch();


    return quote;

  }


};
