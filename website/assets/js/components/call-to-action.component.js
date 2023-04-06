/**
 * <call-to-action>
 * -----------------------------------------------------------------------------
 * A customizeable call to action.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('callToAction', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'title', // Required: The title of this call-to-action
    'text', // Required: The text of the call to action
    'primaryButtonText', // Required: The text of the call to action's button
    'primaryButtonHref', // Required: the url that the call to action button leads
    'secondaryButtonText', // Optional: if provided with a `secondaryButtonHref`, a second button will be added to the call to action with this value as the button text
    'secondaryButtonHref', // Optional: if provided with a `secondaryButtonText`, a second button will be added to the call to action with this value as the href
    'preset',// Optional: if provided, all other values will be ignored, and this component will display a specified varient, can be set to 'premium-upgrade' or 'mdm-beta'.
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    let callToActionTitle = '';
    let callToActionText = '';
    let calltoActionPrimaryBtnText = '';
    let calltoActionPrimaryBtnHref = '';
    let calltoActionSecondaryBtnText = '';
    let calltoActionSecondaryBtnHref = '';
    let callToActionPreset = '';

    return {
      callToActionTitle,
      callToActionText,
      calltoActionPrimaryBtnText,
      calltoActionPrimaryBtnHref,
      calltoActionSecondaryBtnText,
      calltoActionSecondaryBtnHref,
      callToActionPreset
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div id="cta-component">
    <div v-if="!callToActionPreset" purpose="custom-cta">
      <div purpose="custom-cta-content" class="text-white text-center">
        <div purpose="custom-cta-title">{{callToActionTitle}}</div>
        <div purpose="custom-cta-text">{{callToActionText}}</div>
      </div>
      <div purpose="custom-cta-buttons" class="mx-auto d-flex flex-sm-row flex-column justify-content-center">
          <a purpose="primary-button" :class="[ secondaryButtonHref ? 'mr-sm-4 ml-sm-0' : '']" class="text-white d-sm-flex align-items-center justify-content-center btn btn-primary mx-auto":href="calltoActionPrimaryBtnHref">{{calltoActionPrimaryBtnText}}</a>
        <span class="d-flex" v-if="secondaryButtonHref && secondaryButtonText">
          <a purpose="secondary-button" class="btn btn-lg text-white btn-white mr-2 pl-0 mx-auto mx-sm-0 mt-2 mt-sm-0" target="_blank" :href="calltoActionSecondaryBtnHref">{{calltoActionSecondaryBtnText}}</a>
        </span>
      </div>
    </div>
    <div v-else>
      <div v-if="callToActionPreset === 'premium-upgrade'" purpose="fleet-premium-cta" class="d-flex flex-column flex-sm-row align-items-center justify-content-center">
        <div class="order-2 order-sm-1 justify-content-center" purpose="premium-cta-text">
          <h2>Get even more control <br>with <span>Fleet Premium</span></h2>
          <a style="color: #fff; text-decoration: none;" purpose="premium-cta-btn" href="/upgrade">Learn more</a>
        </div>
        <div class="order-1 order-sm-2" purpose="premium-cta-image">
          <img alt="A computer reporting it's disk encryption status" src="/images/premium-landing-feature-4.svg">
        </div>
      </div>
      <div v-else-if="callToActionPreset === 'mdm-beta'" purpose="mdm-beta-cta-container">
        <div purpose="mdm-beta-cta-background">
          <div purpose="mdm-beta-cta" class="d-flex flex-column flex-sm-row">
            <div purpose="mdm-small-banner" class="d-flex d-sm-none">
              <img class="p-0" alt="Fleet city (on a cloud)" src="/images/banner-small-fleet-city-327x257@2x.png">
            </div>
            <div purpose="mdm-cta-text">
              <span purpose="mdm-cta-subtitle">Limited beta</span>
              <span purpose="mdm-cta-title">A better MDM</span>
              <span>Fleet’s cross-platform MDM gives IT teams more visibility out of the box.</span>
              <a style="color: #fff; text-decoration: none;" purpose="mdm-cta-btn" href="/device-management">Request access</a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {
    if(this.preset){
      if(_.contains(['premium-upgrade', 'mdm-beta'], this.preset)){
        this.callToActionPreset = this.preset;
      } else {
        throw new Error('Incomplete usage of <call-to-action>: If providing a type, it must be either \'premium-upgrade\' or \'mdm-beta\'');
      }
    } else {
      if (this.title) {
        this.callToActionTitle = this.title;
      } else {
        throw new Error('Incomplete usage of <call-to-action>: Please provide a `title` example: title="Secure laptops & servers"');
      }
      if (this.text) {
        this.callToActionText = this.text;
      } else {
        throw new Error('Incomplete usage of <call-to-action>: Please provide a `text` example: text="Get up and running with a test environment of Fleet within minutes"');
      }
      if (this.primaryButtonText) {
        this.calltoActionPrimaryBtnText = this.primaryButtonText;
      } else {
        throw new Error('Incomplete usage of <call-to-action>: Please provide a `primaryButtonText`. example: primary-button-text="Get started"');
      }
      if (this.primaryButtonHref) {
        this.calltoActionPrimaryBtnHref = this.primaryButtonHref;
      } else {
        throw new Error('Incomplete usage of <call-to-action>: Please provide a `primaryButtonHref` example: primary-button-href="/get-started?try-it-now"');
      }
      if (this.secondaryButtonText) {
        this.calltoActionSecondaryBtnText = this.secondaryButtonText;
      }
      if (this.secondaryButtonHref) {
        this.calltoActionSecondaryBtnHref = this.secondaryButtonHref;
      }
    }
  },
  watch: {
    title: function(unused) { throw new Error('Changes to `title` are not currently supported in <call-to-action>!'); },
    type: function(unused) { throw new Error('Changes to `type` are not currently supported in <call-to-action>!'); },
    text: function(unused) { throw new Error('Changes to `text` are not currently supported in <call-to-action>!'); },
    primaryButtonText: function(unused) { throw new Error('Changes to `primaryButtonText` are not currently supported in <call-to-action>!'); },
    primaryButtonHref: function(unused) { throw new Error('Changes to `primaryButtonHref` are not currently supported in <call-to-action>!'); },
    secondaryButtonText: function(unused) { throw new Error('Changes to `secondaryButtonText` are not currently supported in <call-to-action>!'); },
    secondaryButtonHref: function(unused) { throw new Error('Changes to `secondaryButtonHref` are not currently supported in <call-to-action>!'); },
  },
  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
  }
});
