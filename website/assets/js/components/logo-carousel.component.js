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
  props: [
    'displayBottomRow'
  ],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      bottomRowVisible: this.displayBottomRow ? true : false,
      isSafariThirteen: bowser.safari && _.startsWith(bowser.version, '13'),
      isIosThirteen: bowser.safari && _.startsWith(bowser.version, '13') && bowser.ios,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div purpose="logos" class="mx-auto d-flex flex-column align-items-center">
    <div purpose="logo-carousel-top">
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/customers">
          <img alt="NVIDIA logo" src="/images/logos/logo-nvidia-110x32@2x.png">
          <img alt="Pinterest logo" src="/images/logos/logo-pinterest-108x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Grafana Labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="LastPass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/customers">
          <img alt="NVIDIA logo" src="/images/logos/logo-nvidia-110x32@2x.png">
          <img alt="Pinterest logo" src="/images/logos/logo-pinterest-108x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Grafana Labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="LastPass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
        </a>
      </div>
      <div purpose="fade-left"></div>
      <div purpose="fade-right"></div>
    </div>
    <div purpose="logo-carousel-bottom" v-if="bottomRowVisible">
      <div purpose="logo-row" class="d-flex flex-row-reverse align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']" >
        <a href="/customers">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="LastPass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="NVIDIA logo" src="/images/logos/logo-nvidia-110x32@2x.png">
          <img alt="Pinterest logo" src="/images/logos/logo-pinterest-108x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Grafana Labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row-reverse align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/customers">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="LastPass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="NVIDIA logo" src="/images/logos/logo-nvidia-110x32@2x.png">
          <img alt="Pinterest logo" src="/images/logos/logo-pinterest-108x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Grafana Labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
        </a>
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
