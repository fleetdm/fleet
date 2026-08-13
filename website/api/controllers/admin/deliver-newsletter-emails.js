module.exports = {


  friendlyName: 'Deliver newsletter emails',


  description: '',


  inputs: {

    emailTemplateName: {
      type: 'string',
      description: ''
    },

    sendToAllSubscribers: {
      type: 'boolean',
      description: 'Whether or not to send the newsletter email to all subscribers'
    },
  },


  exits: {
    success: {
      description: 'The Fleet Newsletter has been sent to subscribers.'
    },
    articleNotFound: {
      description: 'The article that was used to generated the specified email template was not found.'
    },
  },


  fn: async function ({emailTemplateName, sendToAllSubscribers}) {

    // Find the name of the newsletter article.
    let newsletterFileName = emailTemplateName.replace(/^newsletter\/email-/, '');

    // Find the newsletter article the specified email template was generated from in the sails.builtStaticContent configuration.
    let thisNewsletterArticle = _.find(sails.config.builtStaticContent.markdownPages, {sectionRelativeRepoPath: `${newsletterFileName}.md`})

    if(!thisNewsletterArticle) {
      throw 'articleNotFound';
    }
    let numberOfEmailsSent = 0;
    // Build the email subject from the article title.
    let emailSubject = thisNewsletterArticle.meta.articleTitle;
    if(sendToAllSubscribers){

      // Get all active newsletter subscribers.
      let activeNewsletterSubscriptions = await NewsletterSubscription.find({isUnsubscribedFromAll: false});
      await sails.helpers.flow.simultaneouslyForEach(activeNewsletterSubscriptions, async (newsletterSubscriber)=>{


        // Make sure we're not sending duplicate emails to this subscriber.
        let emailsSentToThisSubscriber = newsletterSubscriber.emailsSent;
        if(emailsSentToThisSubscriber.includes(emailTemplateName)) {
          return;
        }


        let deliveredEmail = await sails.helpers.sendTemplateEmail.with({
          to: newsletterSubscriber.emailAddress,
          from: sails.config.custom.newsletterEmailFromAddress,
          fromName: 'Fleet newsletter',
          subject: emailSubject,
          layout: 'layout-email-newsletter',
          template: emailTemplateName,
          templateData: {
            emailAddress: newsletterSubscriber.emailAddress,// Used to build the unsubscribe link.
          },
          ensureAck: true,
        }).tolerate((err)=>{
          sails.log.warn(`When an admin sent the Fleet newsletter to subscribers, an error occured when sending an email to a subscriber (${newsletterSubscriber.emailAddress}). Full error: ${require('util').inspect(err)}`);
          return false;
        });

        // If an email was successfully sent, update the NewsletterSubscription record for this subscriber to ensure they don't receive a duplicate email if this newsletter is sent again.
        if(deliveredEmail) {
          emailsSentToThisSubscriber.push(emailTemplateName);
          await NewsletterSubscription.updateOne({id: newsletterSubscriber.id}).set({emailsSent: emailsSentToThisSubscriber});
          numberOfEmailsSent++;
        }

      })
    } else {
      // Just send the email to the current user's email address.
      await sails.helpers.sendTemplateEmail.with({
        to: this.req.me.emailAddress,
        from: sails.config.custom.newsletterEmailFromAddress,
        fromName: 'Fleet newsletter',
        subject: emailSubject,
        layout: 'layout-email-newsletter',
        template: emailTemplateName,
        templateData: {
          emailAddress: this.req.me.emailAddress,// Used to build the unsubscribe link.
        },
        ensureAck: true,
      })
    }
    // }


    // All done.
    return numberOfEmailsSent;

  }


};
