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
  <div purpose="logos" class="mx-auto d-flex flex-column align-items-center">
    <div purpose="logo-carousel-top">
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/testimonials">
          <!-- Group three -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Dropbox logo" src="/images/logos/logo-dropbox-122x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Leidos logo" src="/images/logos/logo-leidos-102x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Gitlab logo" src="/images/logos/logo-gitlab-111x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Drata logo" src="/images/logos/logo-drata-105x32@2x.png">
          <img alt="Hubspot logo" src="/images/logos/logo-hubspot-113x32@2x.png">
          <!-- Group two -->
          <img alt="Csiro logo" src="/images/logos/logo-csiro-90x32@2x.png">
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Gusto logo" src="/images/logos/logo-gusto-64x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Snowflake logo" src="/images/logos/logo-snowflake-101x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <!-- Group one -->
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Rivian logo" src="/images/logos/logo-rivian-120x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Reddit logo" src="/images/logos/logo-reddit-80x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Deeploi logo" src="/images/logos/logo-deeploi-69x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <!-- Group four -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <img alt="Mozilla logo" src="/images/logos/logo-mozilla-84x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Deloitte logo" src="/images/logos/logo-deloitte-97x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Notion logo" src="/images/logos/logo-notion-68x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">

        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/testimonials">
          <!-- Group three -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Dropbox logo" src="/images/logos/logo-dropbox-122x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Leidos logo" src="/images/logos/logo-leidos-102x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Gitlab logo" src="/images/logos/logo-gitlab-111x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Drata logo" src="/images/logos/logo-drata-105x32@2x.png">
          <img alt="Hubspot logo" src="/images/logos/logo-hubspot-113x32@2x.png">
          <!-- Group two -->
          <img alt="Csiro logo" src="/images/logos/logo-csiro-90x32@2x.png">
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Gusto logo" src="/images/logos/logo-gusto-64x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Snowflake logo" src="/images/logos/logo-snowflake-101x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
          <!-- Group one -->
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Rivian logo" src="/images/logos/logo-rivian-120x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Reddit logo" src="/images/logos/logo-reddit-80x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Deeploi logo" src="/images/logos/logo-deeploi-69x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <!-- Group four -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <img alt="Mozilla logo" src="/images/logos/logo-mozilla-84x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Deloitte logo" src="/images/logos/logo-deloitte-97x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Notion logo" src="/images/logos/logo-notion-68x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
        </a>
      </div>
      <div purpose="fade-left"></div>
      <div purpose="fade-right"></div>
    </div>
    <div purpose="logo-carousel-bottom">
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/testimonials">
          <!-- Group one -->
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Rivian logo" src="/images/logos/logo-rivian-120x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Reddit logo" src="/images/logos/logo-reddit-80x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Deeploi logo" src="/images/logos/logo-deeploi-69x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <!-- Group three -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Dropbox logo" src="/images/logos/logo-dropbox-122x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Leidos logo" src="/images/logos/logo-leidos-102x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Gitlab logo" src="/images/logos/logo-gitlab-111x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Drata logo" src="/images/logos/logo-drata-105x32@2x.png">
          <img alt="Hubspot logo" src="/images/logos/logo-hubspot-113x32@2x.png">
          <!-- Group four -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <img alt="Mozilla logo" src="/images/logos/logo-mozilla-84x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Deloitte logo" src="/images/logos/logo-deloitte-97x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Notion logo" src="/images/logos/logo-notion-68x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <!-- Group two -->
          <img alt="Csiro logo" src="/images/logos/logo-csiro-90x32@2x.png">
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Gusto logo" src="/images/logos/logo-gusto-64x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Snowflake logo" src="/images/logos/logo-snowflake-101x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
        </a>
      </div>
      <div purpose="logo-row" class="d-flex flex-row align-items-center" :class="[isIosThirteen ? 'ios-13-scroll-animation' : isSafariThirteen ? 'safari-13-scroll-animation' : '']">
        <a href="/testimonials">
          <!-- Group one -->
          <img alt="Grafana labs logo" src="/images/logos/logo-grafana-labs-135x32@2x.png">
          <img alt="Rivian logo" src="/images/logos/logo-rivian-120x32@2x.png">
          <img alt="Epic Games logo" src="/images/logos/logo-epic-games-28x32@2x.png">
          <img alt="Reddit logo" src="/images/logos/logo-reddit-80x32@2x.png">
          <img alt="Uber logo" src="/images/logos/logo-uber-65x32@2x.png">
          <img alt="Proton logo" src="/images/logos/logo-proton-95x32@2x.png">
          <img alt="Nutanix logo" src="/images/logos/logo-nutanix-125x32@2x.png">
          <img alt="Mr Beast logo" src="/images/logos/logo-mr-beast-90x32@2x.png">
          <img alt="Deeploi logo" src="/images/logos/logo-deeploi-69x32@2x.png">
          <img alt="Flywire logo" src="/images/logos/logo-flywire-69x32@2x.png">
          <!-- Group three -->
          <img alt="Easygo logo" src="/images/logos/logo-easygo-107x32@2x.png">
          <img alt="Knostic logo" src="/images/logos/logo-knostic-130x32@2x.png">
          <img alt="Dropbox logo" src="/images/logos/logo-dropbox-122x32@2x.png">
          <img alt="Amps logo" src="/images/logos/logo-amps-63x32@2x.png">
          <img alt="Leidos logo" src="/images/logos/logo-leidos-102x32@2x.png">
          <img alt="Prenuvo logo" src="/images/logos/logo-prenuvo-106x32@2x.png">
          <img alt="Gitlab logo" src="/images/logos/logo-gitlab-111x32@2x.png">
          <img alt="Fastly logo" src="/images/logos/logo-fastly-60x32@2x.png">
          <img alt="Drata logo" src="/images/logos/logo-drata-105x32@2x.png">
          <img alt="Hubspot logo" src="/images/logos/logo-hubspot-113x32@2x.png">
          <!-- Group four -->
          <img alt="Censys logo" src="/images/logos/logo-censys-110x32@2x.png">
          <img alt="Faire logo" src="/images/logos/logo-faire-160x32@2x.png">
          <img alt="Bitmex logo" src="/images/logos/logo-bitmex-126x32@2x.png">
          <img alt="Mozilla logo" src="/images/logos/logo-mozilla-84x32@2x.png">
          <img alt="Flock Safety logo" src="/images/logos/logo-flock-safety-154x32@2x.png">
          <img alt="Schodinger logo" src="/images/logos/logo-schodinger-128x32@2x.png">
          <img alt="Deloitte logo" src="/images/logos/logo-deloitte-97x32@2x.png">
          <img alt="Calendly logo" src="/images/logos/logo-calendly-100x32@2x.png">
          <img alt="Notion logo" src="/images/logos/logo-notion-68x32@2x.png">
          <img alt="Lastpass logo" src="/images/logos/logo-lastpass-90x32@2x.png">
          <!-- Group two -->
          <img alt="Csiro logo" src="/images/logos/logo-csiro-90x32@2x.png">
          <img alt="Copia logo" src="/images/logos/logo-copia-98x32@2x.png">
          <img alt="Hadrian logo" src="/images/logos/logo-hadrian-111x32@2x.png">
          <img alt="Gusto logo" src="/images/logos/logo-gusto-64x32@2x.png">
          <img alt="Fireblocks logo" src="/images/logos/logo-fireblocks-142x32@2x.png">
          <img alt="Snowflake logo" src="/images/logos/logo-snowflake-101x32@2x.png">
          <img alt="Vibe logo" src="/images/logos/logo-vibe-72x32@2x.png">
          <img alt="Conductorone logo" src="/images/logos/logo-conductorone-158x32@2x.png">
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
