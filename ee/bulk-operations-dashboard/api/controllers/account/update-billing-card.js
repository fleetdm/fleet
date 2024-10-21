module.exports = {


  friendlyName: 'Update billing card',


  description: 'Update the credit card for the logged-in user.',


  inputs: {

    stripeToken: {
      type: 'string',
      example: 'tok_199k3qEXw14QdSnRwmsK99MH',
      description: 'The single-use Stripe Checkout token identifier representing the user\'s payment source (i.e. credit card.)',
      extendedDescription: 'Omit this (or use "") to remove this user\'s payment source.',
      whereToGet: {
        description: 'This Stripe.js token is provided to the front-end (client-side) code after completing a Stripe Checkout or Stripe Elements flow.'
      }
    },

    billingCardLast4: {
      type: 'string',
      example: '4242',
      description: 'Omit if removing card info.',
      whereToGet: { description: 'Credit card info is provided by Stripe after completing the checkout flow.' }
    },

    billingCardBrand: {
      type: 'string',
      example: 'visa',
      description: 'Omit if removing card info.',
      whereToGet: { description: 'Credit card info is provided by Stripe after completing the checkout flow.' }
    },

    billingCardExpMonth: {
      type: 'string',
      example: '08',
      description: 'Omit if removing card info.',
      whereToGet: { description: 'Credit card info is provided by Stripe after completing the checkout flow.' }
    },

    billingCardExpYear: {
      type: 'string',
      example: '2023',
      description: 'Omit if removing card info.',
      whereToGet: { description: 'Credit card info is provided by Stripe after completing the checkout flow.' }
    },

  },


  fn: async function ({stripeToken, billingCardLast4, billingCardBrand, billingCardExpMonth, billingCardExpYear}) {

    // Add, update, or remove the default payment source for the logged-in user's
    // customer entry in Stripe.
    var stripeCustomerId = await sails.helpers.stripe.saveBillingInfo.with({
      stripeCustomerId: this.req.me.stripeCustomerId,
      token: stripeToken || '',
    }).timeout(5000).retry();

    // Update (or clear) the card info we have stored for this user in our database.
    // > Remember, never store complete card numbers-- only the last 4 digits + expiration!
    // > Storing (or even receiving) complete, unencrypted card numbers would require PCI
    // > compliance in the U.S.
    await User.updateOne({ id: this.req.me.id })
    .set({
      stripeCustomerId,
      hasBillingCard: stripeToken ? true : false,
      billingCardBrand: stripeToken ? billingCardBrand : '',
      billingCardLast4: stripeToken ? billingCardLast4 : '',
      billingCardExpMonth: stripeToken ? billingCardExpMonth : '',
      billingCardExpYear: stripeToken ? billingCardExpYear : ''
    });

  }


};
