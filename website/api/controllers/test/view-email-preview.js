module.exports = {


  friendlyName: 'View email preview',


  description: 'Display "Email preview" page.',

  inputs: {

    template: {
      description: 'The path to an email template, specified in precisely the same way as the equivalent input of the sendTemplateEmail() helper.',
      example: 'email-reset-password',
      type: 'string',
      required: true
    },

    raw: {
      description: 'Whether to return the raw HTML for the email with no JS/CSS (rather than a personalized previewer web page.)',
      extendedDescription: 'This can be used from an iframe to allow for accurately previewing email templates without worrying about style interference from the rest of the Sails app.',
      type: 'boolean',
    }

  },

  exits: {

    success: {
      viewTemplatePath: 'pages/test/email-preview'
    },

    sendRawHtmlInstead: {
      statusCode: 200,
      outputType: 'string',
      outputDescription: 'The raw HTML for the email as a string.',
    },

  },


  fn: async function ({template, raw}) {

    var path = require('path');
    var url = require('url');
    var util = require('util');
    var moment = require('moment-timezone');

    // Determine appropriate email layout and fake data to use.
    let layout = 'layout-email';
    let fakeData;
    switch (template) {
      case 'internal/email-contact-form':
        layout = false;
        fakeData = {
          contactFirstName: 'Sage',
          contactLastName: 'Scorpion',
          contactEmail: 'sage@example.com',
          topic: 'Pricing question',
          message: 'What is the difference between the "Individual" plan and the "Professional" plan?',
        };
        break;
      case 'email-reset-password':
        fakeData = {
          firstName: 'Sage',
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-verify-account':
        fakeData = {
          firstName: 'Sage',
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-verify-new-email':
        fakeData = {
          firstName: 'Sage',
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-receipt':
        fakeData = {
          billingCardBrand: 'MasterCard',
          billingCardLast4: '1234',
          actuallyChargedAt: 1590614058742,
          tz: 'America/New_York',
          lineItems: [
            {
              summary:'Yoga for beginners: healthy alignment',
              amount: 12
            },
            {
              summary:'Yoga for gurus: healthy levitation',
              amount: 14
            }
          ],
        };
        break;
      case 'email-reminder-upcoming-appointment':
        fakeData = {
          firstName: 'Sage', //patron attending the class
          titleAtBooking: 'Yoga for beginners: healthy alignment',
          startsAt: 1590622153990,
          tz: 'America/Chicago',
          token: 'faketoken123',
          host: {
            firstName: 'Jane',
            lastName: 'Williamson',
          },
        };
        break;
      case 'email-share-offering':
        fakeData = {
          formattedDateOfUpcomingEvent: 'Tuesday, March 15th,', //sender's first name
          formattedTimeOfUpcomingEvent: '12:30 PM PT / 3:30 PM EDT', //sender's last name
          linkToEventSignup: 'example.com',
        };
        break;
      default:
        throw new Error(`Unrecognized email template: ${template}`);
    }

    // Compile HTML template using the appropriate layout.
    // > Note that we set the layout, provide access to core `url` package (for
    // > building links and image srcs, etc.), and also provide access to core
    // > `util` package (for dumping debug data in internal emails).
    let emailTemplatePath = path.join('emails/', template);
    if (layout) {
      layout = path.relative(path.dirname(emailTemplatePath), path.resolve('layouts/', layout));
    } else {
      layout = false;
    }

    let sampleHtml = await sails.renderView(
      emailTemplatePath,
      Object.assign({layout, url, util, moment, _ }, fakeData)
    )
    .intercept((err)=>{
      err.message = 'Whoops, that email template failed to render.  Could there be some fake data missing for this particular template in the `switch` statement api/controllers/admin/view-email-template-preview.js?  Any chance you need to re-lift the app after making backend changes?\nMore details: '+err.message;
      return err;
    });

    if (raw) {
      // Respond with raw, rendered HTML for this email:
      throw {sendRawHtmlInstead: sampleHtml};
    } else {
      // Respond with the previewer page for this email:
      return {
        sampleHtml,
        template,
        fakeData,
      };
    }


  }


};
