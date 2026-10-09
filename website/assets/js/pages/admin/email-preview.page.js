parasails.registerPage('email-preview', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    //…
    preview: 'Responsive',
    showNewsletterButtons: false,
    syncing: false,
    cloudError: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
    _.extend(this, SAILS_LOCALS);

    if(_.startsWith(this.template, 'newsletter')) {
      this.showNewsletterButtons = true;
    }
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {


    clickSendTestNewsletter: async function() {
      this.syncing = true;
      await Cloud.deliverNewsletterEmails.with({
        emailTemplateName: this.template, sendToAllSubscribers: false
      }).tolerate((err)=>{
        this.cloudError = err;
        this.syncing = false;
      });
      if(!this.cloudError) {
        this.syncing = false;
      }
    },

    clickSendNewsletterToSubscribers: async function() {
      this.syncing = true;
      let numberOfEmailsSent = await Cloud.deliverNewsletterEmails.with({
        emailTemplateName: this.template,
        sendToAllSubscribers: true
      }).tolerate((err)=>{
        this.cloudError = err;
        this.syncing = false;
      });
      if(!this.cloudError) {
        this.syncing = false;
        window.alert(`Newsletter emails have been sent to ${numberOfEmailsSent} subscribers.`);
      }
    }
  }
});
