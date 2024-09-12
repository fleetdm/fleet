module.exports = {


  friendlyName: 'View email template preview',


  description: 'Display "email template preview" page.',


  urlWildcardSuffix: 'template',


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
      viewTemplatePath: 'pages/admin/email-preview'
    },

    sendRawHtmlInstead: {
      statusCode: 200,
      outputType: 'string',
      outputDescription: 'The raw HTML for the email as a string.',
    },

  },


  fn: async function ({template, raw}) {

    var path = require('path');
    var moment = require('moment');
    var url = require('url');
    var util = require('util');
    // Determine appropriate email layout and fake data to use.
    let layout;
    let fakeData;
    switch (template) {
      case 'internal/email-contact-form':
        layout = false;
        fakeData = {
          contactName: 'Sage',
          contactEmail: 'sage@example.com',
          topic: 'Pricing question',
          message: 'What is the difference between the "Free" plan and the "Premium" plan?',
        };
        break;
      case 'email-reset-password':
        layout = 'layout-email';
        fakeData = {
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-verify-account':
        layout = 'layout-email';
        fakeData = {
          firstName: 'Fleet user',
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-verify-new-email':
        layout = 'layout-email';
        fakeData = {
          fullName: 'Fleet user',
          token: '4-32fad81jdaf$329',
        };
        break;
      case 'email-order-confirmation':
        layout = 'layout-email';
        fakeData = {
          firstName: 'Fleet',
          lastName: 'user',
        };
        break;
      case 'email-subscription-renewal-confirmation':
        layout = 'layout-email';
        fakeData = {
          firstName: 'Fleet',
          lastName: 'user',
        };
        break;
      case 'email-upcoming-subscription-renewal':
        layout = 'layout-email';
        fakeData = {
          firstName: 'Fleet',
          lastName: 'user',
          subscriptionPriceInWholeDollars: 60,
          numberOfHosts: 10,
          subscriptionCostPerHost: 6,
          nextBillingAt: Date.now() + (1000 * 60 * 60 * 24 * 7),
        };
        break;
      case 'email-signed-csr-for-apns':
        layout = 'layout-email';
        fakeData = {};
        break;
      case 'email-sandbox-ready-approved':
        layout = 'layout-email';
        fakeData = {};
        break;
      case 'email-nurture-stage-three':
        layout = 'layout-nurture-email';
        fakeData = {
          firstName: 'Sage',
          emailAddress: 'sage@example.com',
        };
        break;
      case 'email-nurture-stage-four':
        layout = 'layout-nurture-email';
        fakeData = {
          firstName: 'Sage',
          emailAddress: 'sage@example.com',
        };
        break;
      case 'email-nurture-stage-five':
        layout = 'layout-nurture-email';
        fakeData = {
          firstName: 'Sage',
          emailAddress: 'sage@example.com',
        };
        break;
      case 'email-deal-registration':
        layout = 'layout-email';
        fakeData = {
          submittersFirstName: 'Jane',
          submittersLastName: 'Williamson',
          submittersEmailAddress: 'jane@example.com',
          submittersOrganization: 'Fake organization',
          customersFirstName: 'Sage',
          customersLastName: 'Scorpion',
          customersEmailAddress: 'sage@example.com',
          linkedinUrl: 'https://www.linkedin.com/in/sage-scorpion/',
          customersOrganization: 'Fake organization 2',
          customersCurrentMdm: 'Omnissa',
          otherMdmEvaluated: 'Jamf protect',
          preferredHosting: 'Managed cloud',
          expectedDealSize: '$30,000',
          expectedCloseDate: '09/28/2024',
          notes: 'Fake organization 2 is looking for a managed cloud MDM solution with a name that ends with "eet"',
        };
        break;
      case 'email-contact-form':
        fakeData = {
          firstName: 'Jane',
          lastName: 'Williamson',
          emailAddress: 'jane@example.com',
          message: 'Hi, this is a contact form message!',
        };
        break;
      default:
        layout = 'layout-email-newsletter';
        fakeData = {
          emailAddress: 'sage@example.com',
        };
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
      Object.assign({layout, url, util, _, moment }, fakeData)
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
