/**
 * <animated-arrow-button>
 * -----------------------------------------------------------------------------
 * A button with a built-in animated arrow
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('animatedArrowButton', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [
    'buttonType',
    'arrowColor',// For customizing the color of the animated arrow. e.g., arrow-color="#FFF"
    'textColor',// For customizing the color of the button text. e.g., text-color="#FFF"
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      type: 'secondary',
      strokeColor: this.arrowColor ? this.arrowColor : '#FF5C83',
      fontColor: this.textColor ? this.textColor : '#192147'
      // FUTURE: support more button types (primary and secondary)
      // type: this.buttonType && this.buttonType === 'primary' ? 'primary' : 'secondary',
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <a :class="type" style="text-decoration: none;">
    <span purpose="button-text" :style="'color: '+fontColor+';'"><slot name="default"></slot></span>
    <svg purpose="animated-arrow" :style="'stroke: '+strokeColor+';'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12">
      <path purpose="arrow-line" d="M1 6H9" stroke-width="2" stroke-linecap="round"/>
      <path purpose="chevron" d="M1.35712 1L5.64283 6L1.35712 11" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </a>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){
    //…
  },
  beforeDestroy: function() {
    //…
  },
  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

  }
});
