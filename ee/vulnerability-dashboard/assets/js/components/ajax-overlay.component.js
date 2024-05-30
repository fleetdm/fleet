/**
 * <ajax-overlay>
 * -----------------------------------------------------------------------------
 *
 * @type {Component}
 *
 * --- SLOTS: ---
 * N/A
 *
 * --- EVENTS EMITTED: ---
 * N/A
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('ajaxOverlay', {

  //  ╔═╗╦ ╦╔╗ ╦  ╦╔═╗  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝║ ║╠╩╗║  ║║    ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╚═╝╚═╝╩═╝╩╚═╝  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'syncing',
    'syncingMessage'
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╦╔╗╔╔╦╗╔═╗╦═╗╔╗╔╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ║║║║ ║ ║╣ ╠╦╝║║║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╩╝╚╝ ╩ ╚═╝╩╚═╝╚╝╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function () {
    return {
      //…
    };
  },

  beforeMount: function() {
    //…
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div style="top: 0; left: 0; background-color: rgba(255,255,255,0.75); z-index: 4000;" class="position-fixed w-100 h-100" :class="syncing? '' : 'd-none'">
    <div class="row h-100">
      <div class="col h-100"></div>
      <div class="col-6 col-md-4 h-100">
        <div class="d-flex flex-column h-100 justify-content-center">
          <div class="py-5 text-center">
            <h2 purpose="syncing-message" class="text-dark" v-if="syncingMessage">{{syncingMessage}}</h2>
            <span purpose="loading-indicator">
              <span class="loading-dot dot1"></span>
              <span class="loading-dot dot2"></span>
              <span class="loading-dot dot3"></span>
              <span class="loading-dot dot4"></span>
            </span>
          </div>
        </div>
      </div>
      <div class="col h-100"></div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  mounted: async function () {
    //…
  },

  beforeDestroy: function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    //  ╦╔╗╔╔╦╗╔═╗╦═╗╔╗╔╔═╗╦    ╔═╗╦  ╦╔═╗╔╗╔╔╦╗  ╦ ╦╔═╗╔╗╔╔╦╗╦  ╔═╗╦═╗╔═╗
    //  ║║║║ ║ ║╣ ╠╦╝║║║╠═╣║    ║╣ ╚╗╔╝║╣ ║║║ ║   ╠═╣╠═╣║║║ ║║║  ║╣ ╠╦╝╚═╗
    //  ╩╝╚╝ ╩ ╚═╝╩╚═╝╚╝╩ ╩╩═╝  ╚═╝ ╚╝ ╚═╝╝╚╝ ╩   ╩ ╩╩ ╩╝╚╝═╩╝╩═╝╚═╝╩╚═╚═╝

    //…

    //  ╔═╗╦ ╦╔╗ ╦  ╦╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝║ ║╠╩╗║  ║║    ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╚═╝╚═╝╩═╝╩╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝
    // > Public methods are rarely exposed by Vue components, but sometimes they
    // > are an important escape hatch.  They are callable via something like
    // > `this.$refs.ajaxOverlay.doSomething())`, and, by convention, are always
    // > prefixed with "do".
    // N/A

    //  ╔═╗╦═╗╦╦  ╦╔═╗╔╦╗╔═╗  ╔╦╗╔═╗╔╦╗╦ ╦╔═╗╔╦╗╔═╗
    //  ╠═╝╠╦╝║╚╗╔╝╠═╣ ║ ║╣   ║║║║╣  ║ ╠═╣║ ║ ║║╚═╗
    //  ╩  ╩╚═╩ ╚╝ ╩ ╩ ╩ ╚═╝  ╩ ╩╚═╝ ╩ ╩ ╩╚═╝═╩╝╚═╝

    //…

  }

});
