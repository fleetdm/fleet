/**
 * Quote.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝
    organization: { // Note: the current organization exists on the user model, this reflects the organization at the time the quote was created.
      type: 'string',
      description: 'The organization the user entered when they generated a quote',
    },

    numberOfHosts: {
      type: 'number',
      description: 'The number of hosts the user wants a license for',
      required: true,
    },

    quotedPrice: {
      type: 'number',
      description: 'The price of the Fleet Premium license subscription that was generated',
      required: true,
    },
    // TODO: Remove
    // status: {
    //   type: 'string',
    //   description: 'The status of this quote',
    //   isIn: [
    //     'Quote generated', // The user generated the quote
    //     'Quote updated', // The user updated the quote
    //     'Trial active', // The user started a trial after generating a quote
    //     'Inactive', // This quote has no active trial or subscription associated with it
    //     'Did not continue after trial', // The license key associated with this quote did not subscribe after the free trial
    //     'Subscription active' // The license key associated with quote has an active subscription
    //   ],
    //   defaultsTo: 'Quote generated',
    // },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝
    user: {
      model: 'User',
      required: true,
      description: 'The user who created this quote, if they created an account.'
    },
  },

};

