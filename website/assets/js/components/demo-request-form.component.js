/**
 * <demo-request-form>
 * -----------------------------------------------------------------------------
 * An inline demo request form that uses the same submission logic as the
 * "Talk to us" form on /contact. Drop it at the bottom of a page: "Get a demo"
 * links on that page scroll to it (see scroll-to-demo-form.js).
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('demoRequestForm', {
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

  template: `
    <div purpose="demo-request-section" id="demo-form">
      <div purpose="demo-request-intro">
        <h2>See Fleet on your own devices</h2>
        <p>Get Fleet up and running and see what it can do in your environment.</p>
      </div>
      <div purpose="demo-request-card">
        <ajax-form :handle-submitting="handleSubmittingDemoForm" :form-errors.sync="formErrors" :form-data="formData" :form-rules="demoFormRules" :syncing.sync="syncing" :cloud-error.sync="cloudError">
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

  methods: {
    handleSubmittingDemoForm: async function(argins) {
      this.syncing = true;
      window.location = await parasails.util.submitDemoRequest(argins);
    },
  },
});
