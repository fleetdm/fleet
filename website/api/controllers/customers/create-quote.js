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

    notFound: {
      responseType: 'notFound'
    },

    forbidden: {
      responseType: 'forbidden'
    }
  },


  fn: async function ({ numberOfHosts }) {

    let price = numberOfHosts * 12;

    let quote = await Quote.create({
      numberOfHosts: numberOfHosts,
      quotedPrice: price,
      organization: this.req.me.organization,
      user: this.req.me.id,
    }).fetch();


    return quote;

  }


};
