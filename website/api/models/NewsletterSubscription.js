/**
 * NewsletterSubscription.js
 *
 * @description :: A model definition represents a database table/collection.
 * @docs        :: https://sailsjs.com/docs/concepts/models-and-orm/models
 */

module.exports = {

  attributes: {

    //  ╔═╗╦═╗╦╔╦╗╦╔╦╗╦╦  ╦╔═╗╔═╗
    //  ╠═╝╠╦╝║║║║║ ║ ║╚╗╔╝║╣ ╚═╗
    //  ╩  ╩╚═╩╩ ╩╩ ╩ ╩ ╚╝ ╚═╝╚═╝

    emailAddress: {
      type: 'string',
      description: 'The email address that was provided when this subscription was created',
      required: true,
      unique: true,
      isEmail: true,
      maxLength: 200,
      example: 'mary.sue@example.com'
    },

    isUnsubscribedFromAll: {
      type: 'boolean',
      description: 'Whether this newsletter subscription has been updated to indicate a preference for unsubscribing from all current and future newsletters.',
    },

    isSubscribedToReleases: {
      type: 'boolean',
      description: 'Whether the email address associated with this newsletter subscription will be sent release posts and security update emails'
    },

    // isSubscribedToProductArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the product category'
    // },

    // isSubscribedToEngineeringArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the engineering category'
    // },

    // isSubscribedToSecurityArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the security category'
    // },

    // isSubscribedToGuideArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the guides category'
    // },

    // isSubscribedToAnnouncementArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the announcement category'
    // },

    // isSubscribedToDeployArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the deployment guides category'
    // },

    // isSubscribedToPodcastArticles: {
    //   type: 'boolean',
    //   description: 'Whether the email address associated with this newsletter subscription will be sent articles in the podcast category'
    // },

    //  ╔═╗╔╦╗╔╗ ╔═╗╔╦╗╔═╗
    //  ║╣ ║║║╠╩╗║╣  ║║╚═╗
    //  ╚═╝╩ ╩╚═╝╚═╝═╩╝╚═╝


    //  ╔═╗╔═╗╔═╗╔═╗╔═╗╦╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
    //  ╠═╣╚═╗╚═╗║ ║║  ║╠═╣ ║ ║║ ║║║║╚═╗
    //  ╩ ╩╚═╝╚═╝╚═╝╚═╝╩╩ ╩ ╩ ╩╚═╝╝╚╝╚═╝

  },

};

