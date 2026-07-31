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
          <!-- Group three (7 logos) -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">

          <!-- Group two (8 logos) -->
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <img alt="League One Vollyball logo" src="/images/logos/logo-league-one-vollyball-101x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">

          <!-- Group one (7 logos)-->
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="SandboxAQ logo" src="/images/logos/logo-sandboxaq-132x24@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">

          <!-- Group four (8 logos) -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Smarter Technologies logo" src="/images/logos/logo-smarter-technology-130x32@2x.png">
          <img alt="Treeline logo" src="/images/logos/logo-treeline-128x32@2x.png">

        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/customers">
          <!-- Group three (7 logos) -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">

          <!-- Group two (8 logos) -->
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <img alt="League One Vollyball logo" src="/images/logos/logo-league-one-vollyball-101x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">

          <!-- Group one (7 logos)-->
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="SandboxAQ logo" src="/images/logos/logo-sandboxaq-132x24@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">

          <!-- Group four (8 logos) -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Smarter Technologies logo" src="/images/logos/logo-smarter-technology-130x32@2x.png">
          <img alt="Treeline logo" src="/images/logos/logo-treeline-128x32@2x.png">
        </a>
      </div>
      <div purpose="fade-left"></div>
      <div purpose="fade-right"></div>
    </div>
    <div purpose="logo-carousel-bottom" v-if="bottomRowVisible">
      <div purpose="logo-row" class="d-flex flex-row-reverse align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']" >
        <a href="/customers">
          <!-- Group two (8 logos) -->
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <img alt="League One Vollyball logo" src="/images/logos/logo-league-one-vollyball-101x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <!-- Group three (7 logos) -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <!-- Group four (8 logos) -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Smarter Technologies logo" src="/images/logos/logo-smarter-technology-130x32@2x.png">
          <img alt="Treeline logo" src="/images/logos/logo-treeline-128x32@2x.png">
          <!-- Group one (7 logos)-->
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="SandboxAQ logo" src="/images/logos/logo-sandboxaq-132x24@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row-reverse align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/customers">
          <!-- Group two (8 logos) -->
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <img alt="League One Vollyball logo" src="/images/logos/logo-league-one-vollyball-101x32@2x.png">
          <img alt="Cursor logo" src="/images/logos/logo-cursor-103x32@2x.png">
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <!-- Group three (7 logos) -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Abridge logo" src="/images/logos/logo-abridge-133x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <!-- Group four (8 logos) -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <img alt="Smarter Technologies logo" src="/images/logos/logo-smarter-technology-130x32@2x.png">
          <img alt="Treeline logo" src="/images/logos/logo-treeline-128x32@2x.png">
          <!-- Group one (7 logos)-->
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <img alt="SandboxAQ logo" src="/images/logos/logo-sandboxaq-132x24@2x.png">
          <img alt="Webflow logo" src="/images/logos/logo-webflow-144x32@2x.png">
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
