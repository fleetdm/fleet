/**
 * <demo-request-form>
 * -----------------------------------------------------------------------------
 * An inline demo request form that uses the same submission logic as the
 * "Talk to us" form on /contact. While this component is on a page, clicking
 * any "Get a demo" link (including the ones in the site header) jumps to this
 * form instead of navigating to /contact.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('demoRequestForm', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function() {
    return {
      syncing: false,
      formData: { /* … */ },
      formErrors: { /* … */ },
      cloudError: '',
      demoFormRules: {
        emailAddress: { isEmail: true, required: true },
        firstName: { required: true },
        lastName: { required: true },
        organization: { required: true },
        numberOfHosts: { required: true },
      },
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
    <div purpose="demo-request-section" id="demo-form">
      <div purpose="demo-request-intro">
        <h2>See Fleet on your own devices</h2>
        <p>Get Fleet up and running and see what it can do in your environment.</p>
      </div>
      <div purpose="demo-request-card">
        <ajax-form action="deliverTalkToUsFormSubmission" :form-errors.sync="formErrors" :form-data="formData" :form-rules="demoFormRules" :syncing.sync="syncing" :cloud-error.sync="cloudError" @submitted="submittedDemoForm">
          <div purpose="demo-request-grid">
            <div purpose="demo-request-field" class="full-width">
              <label for="demo-email">Work email</label>
              <input class="form-control" id="demo-email" type="email" :class="[formErrors.emailAddress ? 'is-invalid' : cloudError === 'invalidEmailDomain' ? 'is-invalid' : '']" v-model.trim="formData.emailAddress" autocomplete="email">
              <div class="invalid-feedback" v-if="formErrors.emailAddress">Please enter a valid work email address.</div>
              <cloud-error class="p-0 mt-1 mb-0 bg-transparent invalid-feedback" v-if="cloudError === 'invalidEmailDomain'">
                <p>Please enter a valid work email address.</p>
              </cloud-error>
            </div>
            <div purpose="demo-request-field">
              <label for="demo-first-name">First name</label>
              <input class="form-control" id="demo-first-name" type="text" :class="[formErrors.firstName ? 'is-invalid' : '']" v-model.trim="formData.firstName" autocomplete="given-name">
              <div class="invalid-feedback" v-if="formErrors.firstName">Please enter your first name.</div>
            </div>
            <div purpose="demo-request-field">
              <label for="demo-last-name">Last name</label>
              <input class="form-control" id="demo-last-name" type="text" :class="[formErrors.lastName ? 'is-invalid' : '']" v-model.trim="formData.lastName" autocomplete="family-name">
              <div class="invalid-feedback" v-if="formErrors.lastName">Please enter your last name.</div>
            </div>
            <div purpose="demo-request-field">
              <label for="demo-company">Organization</label>
              <input class="form-control" id="demo-company" type="text" :class="[formErrors.organization ? 'is-invalid' : '']" v-model.trim="formData.organization" autocomplete="organization">
              <div class="invalid-feedback" v-if="formErrors.organization">Please enter the name of your organization.</div>
            </div>
            <div purpose="demo-request-field">
              <label for="demo-num-devices">Number of devices</label>
              <input class="form-control" id="demo-num-devices" type="number" min="1" :class="[formErrors.numberOfHosts ? 'is-invalid' : '']" v-model.trim="formData.numberOfHosts">
              <div class="invalid-feedback" v-if="formErrors.numberOfHosts">Please enter the number of devices you manage.</div>
            </div>
          </div>
          <cloud-error v-if="cloudError && cloudError !== 'invalidEmailDomain'"></cloud-error>
          <ajax-button purpose="demo-request-submit" type="submit" syncing-message="Looking up availability…" :syncing="syncing" class="btn btn-primary">Pick a time</ajax-button>
        </ajax-form>
      </div>
    </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function(){
    $(document).on('click.demoRequestForm', 'a[href^="/contact?talkToUs"], a[href="#demo-form"]', this.clickDemoLink);
  },
  beforeDestroy: function() {
    $(document).off('.demoRequestForm');
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickDemoLink: function(event) {
      // Let modified clicks (e.g. cmd+click to open in a new tab) through to /contact.
      if(event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      event.preventDefault();
      $('#navbarToggleExternalContent').collapse('hide');// Close the mobile nav if it's open.
      // Jump to the page's lead-in heading above the form when there is one close by, so the heading stays in view.
      let target = this.$el;
      let leadIn = _.last(document.querySelectorAll('[purpose="bottom-cta"]'));
      let maxLeadInDistance = 800;
      if(leadIn && (leadIn.compareDocumentPosition(this.$el) & Node.DOCUMENT_POSITION_FOLLOWING) && this.$el.getBoundingClientRect().top - leadIn.getBoundingClientRect().top < maxLeadInDistance) {
        target = leadIn;
      }
      let header = document.querySelector('[purpose="header-container"]');
      let headerHeight = header ? header.getBoundingClientRect().height : 0;
      let breathingRoom = 24;
      window.scrollTo({top: target.getBoundingClientRect().top + window.pageYOffset - headerHeight - breathingRoom, behavior: 'auto'});
      this.$el.querySelector('input').focus({preventScroll: true});
    },

    submittedDemoForm: async function(report) {
      this.syncing = true;
      if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 18587089 }); }// eslint-disable-line camelcase

      // Look at result from talking to api and decide what event to track.
      if(report.icp){
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo__icp'); }
        if(typeof window.lintrk !== 'undefined') { window.lintrk('track', { conversion_id: 27493081 }); } // eslint-disable-line camelcase
      } else {
        if(typeof gtag !== 'undefined'){ gtag('event','fleet_website__contact_forms__demo'); }
      }

      // Additional conversion tracking
      if(typeof gtag !== 'undefined'){
        gtag('event', 'conversion', {
          'send_to': 'AW-10788733823/aNrhCNSYrPobEP-GvJgo',
          'value': 1.0,
          'currency': 'USD'
        });
      }//ﬁ

      if(typeof qualified !== 'undefined') {
        qualified('saveFormData',
        {
          email: this.formData.emailAddress,
          name: this.formData.firstName +' '+ this.formData.lastName,
          company: this.formData.organization,
          how_many_hostsdevices_do_you_want_to_manage: this.formData.numberOfHosts,// eslint-disable-line camelcase
        });
        qualified('showFormExperience', 'experience-1772126772950');
      }

      this.goto(report.eventUrl);
    },
  }
});
