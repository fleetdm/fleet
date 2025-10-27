/**
 * <signup-modal>
 * -----------------------------------------------------------------------------
 * A button with a built-in animated arrow
 *
 * @type {Component}
 *
 *
 * @event close   [emitted when the closing process begins]
 * @event opened  [emitted when the opening process is completely done]
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('signupModal', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    // No props.
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      formToDisplay: 'signup',
      syncing: false,
      signupFormData: {},
      signupFormErrors: {},
      signupFormRules: {
        firstName: {required: true},
        lastName: {required: true},
        emailAddress: {required: true, isEmail: true},
        password: {required: true},
      },
      cloudError: undefined,
      loginFormData: {},
      loginFormErrors: {},
      loginFormRules: {
        emailAddress: {required: true, isEmail: true},
        password: {required: true},
      },
      _bsModalIsAnimatingOut: false,

      originalScrollPosition: undefined,//« more on this below
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <transition name="modal" v-on:leave="leave" v-bind:css="false">
    <div class="modal" id="signupmodal" tabindex="-1" role="dialog" data-dismiss="modal" data-target="#signupmodal"  @click="$emit('close')">
      <div class="petticoat"></div>
      <div class="modal-dialog custom-width position-relative" role="document" purpose="modal-dialog" >
        <div class="modal-content" purpose="modal-content" >
          <button type="button" class="position-absolute" data-dismiss="modal" data-target="#signupmodal" aria-label="Close" purpose="modal-close-button" @click="$emit('close')">&times;</button>
          <div purpose="signup-modal">
            <div purpose="modal-logo">
              <img alt="Fleet logo" src="/images/logo-brandmark-32x32@2x.png">
            </div>
          <h3 class="mb-0">Welcome to Fleet</h3>
          <p class="mb-0">We just need a few details in order to get started.</p>
          <div purpose="form-switch" class="d-flex flex-column">
            <label purpose="form-option" class="form-control" :class="[formToDisplay === 'login' ? 'selected' : '']">
              <input type="radio" v-model.trim="formToDisplay" value="login">
              <span purpose="custom-radio"><span purpose="custom-radio-selected"></span></span>
              I have an account
            </label>
            <label purpose="form-option" class="form-control" :class="[formToDisplay === 'signup' ? 'selected' : '']">
              <input type="radio" v-model.trim="formToDisplay" value="signup">
              <span purpose="custom-radio"><span purpose="custom-radio-selected"></span></span>
              I don't have an account
            </label>
          </div>
          <ajax-form action="signup" purpose="modal-form" class="self-service-register" :syncing.sync="syncing" :cloud-error.sync="cloudError" :form-errors.sync="signupFormErrors" :form-data="signupFormData" :form-rules="signupFormRules" @submitted="submittedSignupForm()" v-if="formToDisplay === 'signup'">
            <div class="form-group">
              <label for="email-address">Work email *</label>
              <input tabindex="1" class="form-control" id="email-address"  :class="[signupFormErrors.emailAddress ? 'is-invalid' : '']" v-model.trim="signupFormData.emailAddress" @input="typeClearOneFormError('emailAddress')">
              <div class="invalid-feedback" v-if="signupFormErrors.emailAddress" focus-first>This doesn’t appear to be a valid email address</div>
            </div>
            <div class="form-group">
              <label for="password">Choose a password *</label>
              <input tabindex="2" class="form-control" id="password" type="password"  :class="[signupFormErrors.password ? 'is-invalid' : '']" v-model.trim="signupFormData.password" autocomplete="new-password" @input="typeClearOneFormError('password')">
              <div class="invalid-feedback" v-if="signupFormErrors.password === 'minLength'">Password too short.</div>
              <div class="invalid-feedback" v-if="signupFormErrors.password === 'required'">Please enter a password.</div>
              <p class="mt-2 small"> Minimum length is 8 characters</p>
            </div>
            <div class="row">
              <div class="col-12 col-sm-6 pr-sm-2">
                <div class="form-group">
                  <label for="first-name">First name *</label>
                  <input tabindex="4" class="form-control" id="first-name" type="text"  :class="[signupFormErrors.firstName ? 'is-invalid' : '']" v-model.trim="signupFormData.firstName" autocomplete="first-name" @input="typeClearOneFormError('firstName')">
                  <div class="invalid-feedback" v-if="signupFormErrors.firstName">Please enter your first name.</div>
                </div>
              </div>
              <div class="col-12 col-sm-6 pl-sm-2">
                <div class="form-group">
                  <label for="last-name">Last name *</label>
                  <input tabindex="5" class="form-control" id="last-name" type="text"  :class="[signupFormErrors.lastName ? 'is-invalid' : '']" v-model.trim="signupFormData.lastName" autocomplete="last-name" @input="typeClearOneFormError('lastName')">
                  <div class="invalid-feedback" v-if="signupFormErrors.lastName">Please enter your last name.</div>
                </div>
              </div>
            </div>
            <cloud-error v-if="cloudError==='emailAlreadyInUse'">
              <p>This email is already linked to a Fleet account.<br> Please <a @click="formToDisplay = 'login'">sign in</a> with your email and password.</p>
            </cloud-error>
            <cloud-error v-if="cloudError === 'invalidEmailDomain'">
              <p>
                Please enter your work or school email address.
              </p>
            </cloud-error>
            <blockquote purpose="tip" v-if="cloudError === 'invalidEmailDomain'">
              <img src="/images/icon-info-16x16@2x.png" alt="An icon indicating that this section has important information">
              <div class="d-block">
                <p>Don’t have a work email? Try a local demo of <a href="/try-fleet">Fleet Free</a> instead.</p>
              </div>
            </blockquote>
            <cloud-error purpose="cloud-error" v-if="cloudError && !['emailAlreadyInUse', 'invalidEmailDomain'].includes(cloudError)"></cloud-error>
            <p class="small">By signing up you agree to our <a href="/legal/privacy">privacy policy</a> and <a href="/terms">terms of service</a>.</p>
            <ajax-button tabindex="6" purpose="submit-button" spinner="true" type="submit" :syncing="syncing" class="btn btn-block btn-lg btn-primary mt-4" v-if="!cloudError">Agree and continue</ajax-button>
            <ajax-button tabindex="7" purpose="submit-button" type="button" :syncing="syncing" class="btn btn-block btn-lg btn-primary mt-4" v-if="cloudError" @click="clickResetForm()">Try again</ajax-button>
          </ajax-form>
          <ajax-form class="w-100" action="login" purpose="modal-form"  :syncing.sync="syncing" :cloud-error.sync="cloudError" :form-data="loginFormData" :form-rules="loginFormRules" :form-errors.sync="loginFormErrors" @submitted="submittedLoginForm()" v-else>
            <div class="form-group">
              <label for="email">Email</label>
              <input tabindex="1" type="email" class="form-control" :class="[loginFormErrors.emailAddress ? 'is-invalid' : '']" v-model.trim="loginFormData.emailAddress" autocomplete="email" focus-first>
              <div class="invalid-feedback" v-if="loginFormErrors.emailAddress">Please provide a valid email address.</div>
            </div>
            <div class="form-group">
              <label for="password">Password</label>
              <input tabindex="2" type="password" class="form-control" :class="[loginFormErrors.password ? 'is-invalid' : '']" v-model.trim="loginFormData.password" autocomplete="current-password">
              <div class="invalid-feedback" v-if="loginFormErrors.password">Please enter your password.</div>
            </div>
            <cloud-error v-if="cloudError === 'noUser'">The email address provided doesn't match an existing account. Create an account <a @click="formToDisplay = 'signup'">here</a>.</cloud-error>
            <cloud-error v-else-if="cloudError === 'badCombo'">Something’s not quite right with your email or password.</cloud-error>
            <cloud-error v-else-if="cloudError"></cloud-error>
            <div class="pb-3">
              <ajax-button tabindex="3" :syncing="syncing" spinner="true" purpose="submit-button" class="btn-primary mt-4 btn-lg btn-block">Sign in</ajax-button>
            </div>
          </ajax-form>
          </div>
        </div><!-- /.modal-content -->
      </div><!-- /.modal-dialog -->
    </div><!-- /.modal -->
  </transition>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    // ^^ Note that this is not an `async function`.
    // This is just to be safe, since the timing here is a little tricky w/ the
    // animations and the fact that we're integrating with Bootstrap's modal.
    // (That said, it might work fine-- just hasn't been extensively tested.)

    // Immediately call out to the Bootstrap modal and tell it to show itself.
    $(this.$el).modal({
      // Set the modal backdrop to the 'static' option, which means it doesn't close the modal
      // when clicked.
      backdrop: 'static',
      show: false
    });

    // Attach listener for underlying custom modal closing event,
    // and when that happens, have Vue emit a custom "close" event.
    // (Note: This isn't just for convenience-- it's crucial that
    // the parent logic can use this event to update its scope.)
    $(this.$el).on('hide.bs.modal', ()=>{

      this._bsModalIsAnimatingOut = true;
      this.$emit('close');

    });//œ

    // Attach listener for underlying custom modal "opened" event,
    // and when that happens, have Vue emit our own custom "opened" event.
    // This is so we know when the entry animation has completed, allows
    // us to do cool things like auto-focus the first input in a form modal.
    $(this.$el).on('shown.bs.modal', ()=>{
      $('.modal-backdrop.show').addClass('signup');
      // Focus our "focus-first" field, if relevant.
      // (but not on mobile, because it can get weird)
      if(typeof bowser !== 'undefined' && !bowser.mobile && this.$find('[focus-first]').length > 0) {
        this.$focus('[focus-first]');
      }
      this.$emit('opened');
      // $(this.$el).off('shown.bs.modal');
    });//ƒ
  },
  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    switchForm(form) {
      this.formToDisplay = form;
    },

    clickResetForm: async function() {
      this.cloudError = '';
      this.signupFormErrors = {};
      await this.forceRender();
    },

    typeClearOneFormError: async function(field) {
      if(this.signupFormErrors[field]){
        this.signupFormErrors = _.omit(this.signupFormErrors, field);
      }
    },


    submittedSignupForm: async function(){
      this.syncing = true;
      this.goto('/try');
    },
    submittedLoginForm: async function() {
      this.syncing = true;
      this.goto('/try');
    },
    leave: function (el, done) {
      // > Note: This function signature comes from Vue.js's transition system.
      // > It will likely be replaced with `async function (el){…}` in a future
      // > release of Vue/Sails.js (i.e. no callback argument).

      // If this shutting down was spawned by the bootstrap modal's built-in logic,
      // then we'll have already begun animating the modal shut.  So we check our
      // spinlock to make sure.  If it turns out that we HAVEN'T started that process
      // yet, then we go ahead and start it now.
      if (!this._bsModalIsAnimatingOut) {
        $(this.$el).modal('hide');
      }//ﬁ

      // When the bootstrap modal finishes animating into nothingness, unbind all
      // the DOM events used by bootstrap, and then call `done()`, which passes
      // control back to Vue and lets it finish the job (i.e. afterLeave).
      //
      // > Note that the other lifecycle events like `destroyed` were actually
      // > already fired at this point.
      // >
      // > Also note that, since we're potentially long past the `destroyed` point
      // > of the lifecycle here, we can't call `.$emit()` anymore either.  So,
      // > for example, we wouldn't be able to emit a "fullyClosed" event --
      // > because by the time it'd be appropriate to emit the Vue event, our
      // > context for triggering it (i.e. the relevant instance of this component)
      // > will no longer be capable of emitting custom Vue events (because by then,
      // > it is no longer "reactive").
      // >
      // > For more info, see:
      // > https://github.com/vuejs/vue-router/issues/1302#issuecomment-291207073
      $(this.$el).on('hidden.bs.modal', ()=>{
        $(this.$el).off('hide.bs.modal');
        $(this.$el).off('hidden.bs.modal');
        $(this.$el).off('shown.bs.modal');
        done();
      });//_∏_

    },
  }
});
