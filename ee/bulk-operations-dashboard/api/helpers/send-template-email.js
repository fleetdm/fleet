module.exports = {


  friendlyName: 'Send template email',


  description: 'Send an email using a template.',


  extendedDescription: 'To ease testing and development, if the provided "to" email address ends in "@example.com", '+
    'then the email message will be written to the terminal instead of actually being sent.'+
    '(Thanks [@simonratner](https://github.com/simonratner)!)',


  inputs: {


    template: {
      description: 'The relative path to an EJS template within our `views/emails/` folder -- WITHOUT the file extension.',
      extendedDescription: 'Use strings like "foo" or "foo/bar", but NEVER "foo/bar.ejs" or "/foo/bar".  For example, '+
        '"internal/email-contact-form" would send an email using the "views/emails/internal/email-contact-form.ejs" template.',
      example: 'email-reset-password',
      type: 'string',
      required: true
    },

    templateData: {
      description: 'A dictionary of data which will be accessible in the EJS template.',
      extendedDescription: 'Each key will be a local variable accessible in the template.  For instance, if you supply '+
        'a dictionary with a \`friends\` key, and \`friends\` is an array like \`[{name:"Chandra"}, {name:"Mary"}]\`),'+
        'then you will be able to access \`friends\` from the template:\n'+
        '\`\`\`\n'+
        '<ul>\n'+
         '<% for (friend of friends){ %><li><%= friend.name %></li><% }); %>\n'+
        '</ul>\n'+
        '\`\`\`'+
        '\n'+
        'This is EJS, so use \`<%= %>\` to inject the HTML-escaped content of a variable, \`<%= %>\` to skip HTML-escaping '+
        'and inject the data as-is, or \`<% %>\` to execute some JavaScript code such as an \`if\` statement or \`for\` loop.',
      type: {},
      defaultsTo: {}
    },

    to: {
      description: 'The email address of the primary recipient.',
      extendedDescription: 'If this is any address ending in "@example.com", then don\'t actually deliver the message. '+
        'Instead, just log it to the console.',
      example: 'nola.thacker@example.com',
      required: true,
      isEmail: true,
    },

    toName: {
      description: 'Name of the primary recipient as displayed in their inbox.',
      example: 'Nola Thacker',
    },

    subject: {
      description: 'The subject of the email.',
      example: 'Hello there.',
      defaultsTo: ''
    },

    from: {
      description: 'An override for the default "from" email that\'s been configured.',
      example: 'anne.martin@example.com',
      isEmail: true,
    },

    fromName: {
      description: 'An override for the default "from" name.',
      example: 'Anne Martin',
    },

    layout: {
      description: 'Set to `false` to disable layouts altogether, or provide the path (relative '+
        'from `views/layouts/`) to an override email layout.',
      defaultsTo: 'layout-email',
      custom: (layout)=>layout===false || _.isString(layout)
    },

    ensureAck: {
      description: 'Whether to wait for acknowledgement (to hear back) that the email was successfully sent (or at least queued for sending) before returning.',
      extendedDescription: 'Otherwise by default, this returns immediately and delivers the request to deliver this email in the background.',
      type: 'boolean',
      defaultsTo: false
    },

    bcc: {
      description: 'The email addresses of recipients secretly copied on the email.',
      example: ['jahnna.n.malcolm@example.com'],
    },

    attachments: {
      description: 'Attachments to include in the email, with the file content encoded as base64.',
      whereToGet: {
        description: 'If you have `sails-hook-uploads` installed, you can use `sails.reservoir` to get an attachment into the expected format.',
      },
      example: [
        {
          contentBytes: 'iVBORw0KGgoAA…',
          name: 'sails.png',
          type: 'image/png',
        }
      ],
      defaultsTo: [],
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Email delivery report',
      outputDescription: 'A dictionary of information about what went down.',
      outputType: {
        loggedInsteadOfSending: 'boolean'
      }
    }

  },


  fn: async function({template, templateData, to, toName, subject, from, fromName, layout, ensureAck, bcc, attachments}) {

    var path = require('path');
    var url = require('url');
    var util = require('util');


    if (!_.startsWith(path.basename(template), 'email-')) {
      sails.log.warn(
        'The "template" that was passed in to `sendTemplateEmail()` does not begin with '+
        '"email-" -- but by convention, all email template files in `views/emails/` should '+
        'be namespaced in this way.  (This makes it easier to look up email templates by '+
        'filename; e.g. when using CMD/CTRL+P in Sublime Text.)\n'+
        'Continuing regardless...'
      );
    }

    if (_.startsWith(template, 'views/') || _.startsWith(template, 'emails/')) {
      throw new Error(
        'The "template" that was passed in to `sendTemplateEmail()` was prefixed with\n'+
        '`emails/` or `views/` -- but that part is supposed to be omitted.  Instead, please\n'+
        'just specify the path to the desired email template relative from `views/emails/`.\n'+
        'For example:\n'+
        '  template: \'email-reset-password\'\n'+
        'Or:\n'+
        '  template: \'admin/email-contact-form\'\n'+
        ' [?] If you\'re unsure or need advice, see https://sailsjs.com/support'
      );
    }//•

    // Determine appropriate email layout and template to use.
    var emailTemplatePath = path.join('emails/', template);
    var emailTemplateLayout;
    if (layout) {
      emailTemplateLayout = path.relative(path.dirname(emailTemplatePath), path.resolve('layouts/', layout));
    } else {
      emailTemplateLayout = false;
    }

    // Compile HTML template.
    // > Note that we set the layout, provide access to core `url` package (for
    // > building links and image srcs, etc.), and also provide access to core
    // > `util` package (for dumping debug data in internal emails).
    var htmlEmailContents = await sails.renderView(
      emailTemplatePath,
      _.extend({layout: emailTemplateLayout, url, util }, templateData)
    )
    .intercept((err)=>{
      err.message =
      'Could not compile view template.\n'+
      '(Usually, this means the provided data is invalid, or missing a piece.)\n'+
      'Details:\n'+
      err.message;
      return err;
    });

    // Sometimes only log info to the console about the email that WOULD have been sent.
    // Specifically, if the "To" email address is anything "@example.com".
    //
    // > This is used below when determining whether to actually send the email,
    // > for convenience during development, but also for safety.  (For example,
    // > a special-cased version of "user@example.com" is used by Trend Micro Mars
    // > scanner to "check apks for malware".)
    var isToAddressConsideredFake = Boolean(to.match(/@example\.com$/i));

    // If that's the case, or if we're in the "test" environment, then log
    // the email instead of sending it:
    var dontActuallySend = (
      sails.config.environment === 'test' || isToAddressConsideredFake
    );
    if (dontActuallySend) {
      sails.log(
        'Skipped sending email, either because the "To" email address ended in "@example.com"\n'+
        'or because the current \`sails.config.environment\` is set to "test".\n'+
        '\n'+
        'But anyway, here is what WOULD have been sent:\n'+
        '-=-=-=-=-=-=-=-=-=-=-=-=-= Email log =-=-=-=-=-=-=-=-=-=-=-=-=-\n'+
        'To: '+to+'\n'+
        'Subject: '+subject+'\n'+
        '\n'+
        'Body:\n'+
        htmlEmailContents+'\n'+
        '-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-'
      );
    } else {
      // Otherwise, we'll check that all required Mailgun credentials are set up
      // and, if so, continue to actually send the email.

      if (!sails.config.custom.sendgridSecret) {
        throw new Error(
          'Cannot deliver email to "'+to+'" because:\n'+
          (()=>{
            let problems = [];
            if (!sails.config.custom.sendgridSecret) {
              problems.push(' • Sendgrid secret is missing from this app\'s configuration (`sails.config.custom.sendgridSecret`)');
            }
            return problems.join('\n');
          })()+
          '\n'+
          'To resolve these configuration issues, add the missing config variables to\n'+
          '\`config/custom.js\`-- or in staging/production, set them up as system\n'+
          'environment vars.  (If you don\'t have a Sendgrid secret, you can\n'+
          'sign up for free at https://sendgrid.com to receive credentials.)\n'+
          '\n'+
          '> Note that, for convenience during development, there is another alternative:\n'+
          '> In lieu of setting up real Sendgrid credentials, you can "fake" email\n'+
          '> delivery by using any email address that ends in "@example.com".  This will\n'+
          '> write automated emails to your logs rather than actually sending them.\n'+
          '> (To simulate clicking on a link from an email, just copy and paste the link\n'+
          '> from the terminal output into your browser.)\n'+
          '\n'+
          '[?] If you\'re unsure, visit https://sailsjs.com/support'
        );
      }

      var subjectLinePrefix = sails.config.environment === 'production' ? '' : sails.config.environment === 'staging' ? '[FROM STAGING] ' : '[FROM LOCALHOST] ';
      var messageData = {
        htmlMessage: htmlEmailContents,
        to: to,
        toName: toName,
        bcc: bcc,
        subject: subjectLinePrefix+subject,
        from: from,
        fromName: fromName,
        attachments
      };

      var deferred = sails.helpers.sendgrid.sendHtmlEmail.with(messageData);
      if (ensureAck) {
        await deferred;
      } else {
        // FUTURE: take advantage of .background() here instead (when available)
        deferred.exec((err)=>{
          if (err) {
            sails.log.error(
              'Background instruction failed:  Could not deliver email:\n'+
              util.inspect({template, templateData, to, toName, subject, from, fromName, layout, ensureAck, bcc, attachments},{depth:null})+'\n',
              'Error details:\n'+
              util.inspect(err)
            );
          } else {
            sails.log.info(
              'Background instruction complete:  Email sent via email delivery service (or at least queued):\n'+
              util.inspect({to, toName, subject, from, fromName, bcc},{depth:null})
            );
          }
        });//_∏_
      }//ﬁ
    }//ﬁ

    // All done!
    return {
      loggedInsteadOfSending: dontActuallySend,
    };

  }

};
