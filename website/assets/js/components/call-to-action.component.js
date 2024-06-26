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
    <div purpose="custom-cta">
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
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function() {
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
