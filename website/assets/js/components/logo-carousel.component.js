/**
 * <logo-carousel>
 * -----------------------------------------------------------------------------
 * A row of logos that scroll infinitly to the left.
 *
 * @type {Component}
 *
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('logoCarousel', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      isSafariThirteen: bowser.safari && _.startsWith(bowser.version, '13'),
      isIosThirteen: bowser.safari && _.startsWith(bowser.version, '13') && bowser.ios,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div purpose="logos" class="mx-auto d-flex flex-row align-items-center">
    <div purpose="logo-carousel">
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <img alt="Notion logo" src="/images/logo-notion-68x32@2x.png">
        <img alt="Gusto logo" src="/images/logo-gusto-64x32@2x.png">
        <img alt="Rivian logo" src="/images/logo-rivian-120x32@2x.png">
        <img alt="Deloitte logo" src="/images/logo-deloitte-97x32@2x.png">
        <img alt="Flywire logo" src="/images/logo-flywire-69x32@2x.png">
        <img alt="Snowflake logo" src="/images/logo-snowflake-101x32@2x.png">
        <img alt="Uber logo" src="/images/logo-uber-65x32@2x.png">
        <img alt="Atlassian logo" src="/images/logo-atlassian-140x32@2x.png">
        <img alt="Toast logo" src="/images/logo-toast-91x32@2x.png">
        <img alt="Fastly logo" src="/images/logo-fastly-60x32@2x.png">
        <img alt="Hashicorp logo" src="/images/logo-hashicorp-103x32@2x.png">
        <img alt="Dropbox logo" src="/images/logo-dropbox-122x32@2x.png">
        <img alt="Reddit logo" src="/images/logo-reddit-80x32@2x.png">
      </div>
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <img alt="Notion logo" src="/images/logo-notion-68x32@2x.png">
        <img alt="Gusto logo" src="/images/logo-gusto-64x32@2x.png">
        <img alt="Rivian logo" src="/images/logo-rivian-120x32@2x.png">
        <img alt="Deloitte logo" src="/images/logo-deloitte-97x32@2x.png">
        <img alt="Flywire logo" src="/images/logo-flywire-69x32@2x.png">
        <img alt="Snowflake logo" src="/images/logo-snowflake-101x32@2x.png">
        <img alt="Uber logo" src="/images/logo-uber-65x32@2x.png">
        <img alt="Atlassian logo" src="/images/logo-atlassian-140x32@2x.png">
        <img alt="Toast logo" src="/images/logo-toast-91x32@2x.png">
        <img alt="Fastly logo" src="/images/logo-fastly-60x32@2x.png">
        <img alt="Hashicorp logo" src="/images/logo-hashicorp-103x32@2x.png">
        <img alt="Dropbox logo" src="/images/logo-dropbox-122x32@2x.png">
        <img alt="Reddit logo" src="/images/logo-reddit-80x32@2x.png">
      </div>
      <div purpose="fade-left"></div>
      <div purpose="fade-right"></div>
    </div>

  </div>
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
    //…
  }
});
