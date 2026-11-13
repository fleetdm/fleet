parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: undefined,
    quotesWithVideoLinks: [],

    // For the hero "Get a demo" email form
    syncing: false,
    heroFormData: {
      submittedFromPage: '/',
    },
    heroFormErrors: {},
    heroCloudError: '',
    heroFormRules: {
      emailAddress: {isEmail: true, required: true},
    },

    // For the testimonials veil
    testimonialsVisibleHeight: 600,
    testimonialsHeightIncrement: 560,// « Roughly one to two rows of cards.
    allTestimonialsVisible: false,

    // For MDM comparison table
    comparisonModeForIt: 'intune',
    comparisonModeFriendlyNames: {
      intune: 'Intune',
      jamf: 'Jamf Pro',
      omnissa: 'Workspace ONE',
      ivanti: 'Ivanti',
      tanium: 'Tanium',
      ansible: 'Ansible',
      puppet: 'Puppet',
      chef: 'Chef',
      rapid: 'Rapid 7',
      crowdstrike: 'Crowdstrike',
      qualys: 'Qualys',
      tenable: 'Tenable',
      defender: 'Defender',
      patchmypc: 'PatchMyPC',
    }
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(window.location.hash === '#unsubscribed'){
      this.modal = 'unsubscribed';
      window.location.hash = '';
    }
    for(let quote of this.testimonialsWithVideoLinks) {
      this.quotesWithVideoLinks.push({
        modalId: _.kebabCase(quote.quoteAuthorName),
        embedId: quote.videoIdForEmbed,
      });
    }
  },
  mounted: async function() {
    $('[data-toggle="tooltip"]').tooltip({
      container: '#homepage',
      trigger: 'hover',
    });
    // If every testimonial already fits, skip the veil and the "Show more" button.
    let masonry = this.$el.querySelector('[purpose="testimonials-masonry"]');
    if(masonry && masonry.scrollHeight <= this.testimonialsVisibleHeight) {
      this.allTestimonialsVisible = true;
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    submittedHeroEmailForm: async function(report) {
      // Same tracking and routing as the <demo-request-form> component. The visitor only gave a work email, so the server's enrichment fills in the rest.
      this.syncing = true;
      if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 18587089 }); }// eslint-disable-line camelcase
      if(report.icp){
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo__icp'); }
        if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 27493081 }); } // eslint-disable-line camelcase
      } else {
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo'); }
      }
      if(typeof gtag !== 'undefined'){
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/aNrhCNSYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }//ﬁ
      if(typeof qualified !== 'undefined') {
        qualified('saveFormData', { email: this.heroFormData.emailAddress });
        qualified('showFormExperience', 'experience-1772126772950');
      }
      this.goto(report.eventUrl);
    },
    clickShowMoreTestimonials: function() {
      // Measured on each click, since the masonry's height changes with the viewport width.
      let fullHeight = this.$el.querySelector('[purpose="testimonials-masonry"]').scrollHeight;
      let nextHeight = this.testimonialsVisibleHeight + this.testimonialsHeightIncrement;
      if(nextHeight >= fullHeight) {
        // Animate to the full height first, then drop the max-height so later resizes can't clip cards.
        this.testimonialsVisibleHeight = fullHeight;
        setTimeout(()=>{
          this.allTestimonialsVisible = true;
        }, 400);
      } else {
        this.testimonialsVisibleHeight = nextHeight;
      }
    },
    clickOpenVideoModal: function(modalName) {
      this.modal = _.kebabCase(modalName);
    },
    clickSwitchComparisonTableColumn: async function(option){
      this.comparisonModeForIt = option;
      await setTimeout(()=>{
        $('[data-toggle="tooltip"]').tooltip({
          container: '#homepage',
          trigger: 'hover',
        });
      }, 250);
    },
    closeModal: function() {
      this.modal = undefined;
    },
  }
});
