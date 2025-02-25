/**
 * <stripe-card-element>
 * -----------------------------------------------------------------------------
 * A wrapper for the Stripe Elements "card" component (https://stripe.com/elements)
 *
 * @type {Component}
 *
 * @event update:busy  [:busy.sync="…"]
 * @event input        [emitted when the stripe token changes (supports v-model)]
 * @event invalidated  [emitted when the field had been changed to include an invalid value]
 * @event validated    [emitted when the field had been changed to include a valid value]
 * -----------------------------------------------------------------------------
 */
parasails.registerComponent('stripeCardElement', {
  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔═╗╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠╣ ╠═╣║  ║╣
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╚  ╩ ╩╚═╝╚═╝
  props: [
    'stripePublishableKey',
    'isErrored',
    'errorMessage',//« optional custom error message to display
    'value',//« the v-model passed in.
    'busy',
    'showExisting',// « whether to show the existing card info passed into `v-model`
  ],
  //  ╔╦╗╔═╗╦═╗╦╔═╦ ╦╔═╗
  //  ║║║╠═╣╠╦╝╠╩╗║ ║╠═╝
  //  ╩ ╩╩ ╩╩╚═╩ ╩╚═╝╩
  template: `
  <div>
    <div v-if="existingCardData">
      <span class="existing-card">{{existingCardData.billingCardBrand}} ending in <strong>{{existingCardData.billingCardLast4}}</strong></span>
      <small class="new-card-text d-inline-block ml-2">(Want to use a different card ? <a class="text-primary change-card-button" type="button" @click="clickChangeExistingCard()">Click here</a>.)</small>
    </div>
    <div class="card-element-wrapper" :class="[existingCardData ? 'secret-card-element-wrapper' : '', isErrored ? 'is-invalid' : '']" :aria-hidden="existingCardData ? true : false">
      <div class="card-element form-control" :class="isErrored ? 'is-invalid' : ''" card-element></div>
      <span class="status-indicator syncing text-primary" :class="[isSyncing ? '' : 'hidden']"><span class="fa fa-spinner"></span></span>
      <span class="status-indicator text-primary" :class="[isValidated ? '' : 'hidden']"><span class="fa fa-check-circle"></span></span>
      <div class="invalid-feedback" v-if="!isValidated && isErrored">{{ errorMessage || 'Please check your card info.'}}</div>
    </div>
  </div>
  `,
  //  ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      isSyncing: false,
      isValidated: false,
      existingCardData: undefined,
      // The underlying Stripe instance
      _stripe: undefined,
      // The underlying Stripe elements instance
      _elements: undefined,
      // The underlying Stripe element instance this component creates as it mounts.
      _element: undefined,
    };
  },
  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Initialize an instance of Stripe "elements", which we'll pass into
    // our <stripe-element> component instances.
    // (We also save an instance of `stripe` for use below.)
    this._stripe = Stripe(this.stripePublishableKey);
    this._elements = this._stripe.elements();
  },
  mounted: function (){
    if(this.showExisting && _.isObject(this.value) && this.value.stripeToken && this.value.billingCardBrand && this.value.billingCardLast4) {
      this.existingCardData = _.extend({}, this.value);
    }
    this._element = this._elements.create('card', {
      // Classes
      // > https://stripe.com/docs/js/elements_object/create_element?type=card#elements_create-options-classes
      classes: {
        // When the iframe is focused, attach the "pseudofocused" class
        // to our wrapper <div>.
        focus: 'pseudofocused'
      },
      // iframe styles:
      // > https://stripe.com/docs/js/appendix/style?type=card
      // You can update this section to match your website's theme
      style: {
        base: {
          lineHeight: '36px',
          fontSize: '16px',
          color: '#495057',
          iconColor: '#14acc2',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
          '::placeholder': {
            color: '#6c757d',
          },
        },
        invalid: {
          color: '#dc3545',
        },
      },
    });
    this._element.mount($(this.$el).find('[card-element]')[0]);
    //  When a change occurs, immediately clear the token by
    //  emitting an input event with '', show a loading spinner, and start
    //  fetching a new token. Then in userland, the validation error for a missing
    //  card becomes something reasonable that implies that we may not have finished
    //  getting it yet, so hold your horses.
    this._element.on('change', (stripeEvent)=> {
      // If there is an error, set the v-model to be empty.
      if(stripeEvent.error) {
        this.$emit('input', '');
      } else if(stripeEvent.complete) {
        // If the field is complete, (aka valid), fetch a token and set that on the v-model
        // (first clearing out the v-model, so this won't be considered valid yet e.g. if it was just changed.
        if(this.isSyncing) { return; }
        this.$emit('');
        this.isSyncing = true;
        this.$emit('update:busy', true);
        this.isValidated = false;
        this._fetchNewToken();
      } else {
        // FUTURE: possibly handle other events, if necessary.
      }//ﬁ
    });//œ
  },
  beforeDestroy: function (){
    // Note: There isn't any documented way to tear down a `stripe` instance.
    // Same thing for the `elements` instance.  Only individual "element" instances
    // can be cleaned up after, using `.unmount()`.
    this._element.unmount();
  },
  //  ╔═╗╦  ╦╔═╗╔╗╔╔╦╗╔═╗
  //  ║╣ ╚╗╔╝║╣ ║║║ ║ ╚═╗
  //  ╚═╝ ╚╝ ╚═╝╝╚╝ ╩ ╚═╝
  methods: {
    clickChangeExistingCard: function() {
      this.existingCardData = undefined;
      this.$emit('input', '');
    },
    // Public method for fetching a fresh token (e.g. if card is declined)
    doGetNewToken: function() {
      this.isSyncing = true;
      this.$emit('update:busy', true);
      this.isValidated = false;
      this.$emit('input', '');
      this._fetchNewToken();
    },
    _fetchNewToken: function() {
      this._getStripeTokenFromCardElement(this._stripe, this._element)
      .then((paymentSourceInfo)=>{
        this.isSyncing = false;
        this.$emit('update:busy', false);
        this.isValidated = true;
        this.$emit('input', paymentSourceInfo);
      }).catch((err)=>{
        this.isSyncing = false;
        this.$emit('update:busy', false);
        this.isValidated = false;
        // This error is only relevant if something COMPLETELY unexpected goes wrong,
        // in which case we want to actually know about that.
        throw err;
      });//_∏_
    },
    _getStripeTokenFromCardElement: function(stripeInstance, stripeElement) {
      // Build a Promise & send it back as our "thenable" (AsyncFunction's return value).
      // (this is necessary b/c we're wrapping an api that isn't `await`-compatible)
      return new Promise((resolve, reject)=>{
        try {
          // Create a stripe token using the Stripe "element".
          stripeInstance.createToken(stripeElement)
          .then((result)=>{
            // Silently ignore the case where the field is empty, or if there are
            // card validation issues.
            if(!result || result.error) {
              resolve();
              return;
            }
            // Send back the token & payment info.
            resolve({
              stripeToken: result.token.id,
              billingCardBrand: result.token.card.brand,
              billingCardLast4: result.token.card.last4,
              billingCardExpMonth: result.token.card.exp_month,
              billingCardExpYear: result.token.card.exp_year
            });
          });
        } catch (err) {
          console.error('Could not obtain Stripe token:', err);
          reject(err);
        }
      });//_∏_
    }
  }
});
