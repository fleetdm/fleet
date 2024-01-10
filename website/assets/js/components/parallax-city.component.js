/**
 * <parallax-city>
 * -----------------------------------------------------------------------------
 * A button with a built-in loading spinner.
 *
 * @type {Component}
 *
 * @event click   [emitted when clicked]
 * -----------------------------------------------------------------------------
 */

parasails.registerComponent('parallaxCity', {
  //  ╔═╗╦═╗╔═╗╔═╗╔═╗
  //  ╠═╝╠╦╝║ ║╠═╝╚═╗
  //  ╩  ╩╚═╚═╝╩  ╚═╝
  props: [],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      //…
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div purpose="parallax-city">
      <div class="layer layer-9" scroll-amount=12></div>
      <div class="layer layer-8" scroll-amount=28></div>
      <div class="layer layer-7" scroll-amount=20></div>
      <div class="layer layer-6" scroll-amount=40></div>
      <div class="layer layer-5" scroll-amount=40></div>
      <div class="layer layer-4" scroll-amount=60></div>
      <div class="layer layer-3" scroll-amount=60></div>
      <div class="layer layer-2" scroll-amount=100></div>
      <div class="layer layer-1" scroll-amount=120></div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function(){
    let parallaxCityElement = document.querySelector('[purpose="parallax-city"]');
    let rect = parallaxCityElement.getBoundingClientRect();
    let isElementCurrentlyVisible = (rect.bottom > (parallaxCityElement.offsetTop + parallaxCityElement.clientHeight));
    if(isElementCurrentlyVisible) {
      this.handleParallaxScroll();
    }
    document.querySelectorAll('div.layer').forEach((layer)=>{
      let initialPosition = layer.getAttribute('scroll-amount');
      layer.style.bottom = `-${Number(initialPosition) + 1}px`;
    });
    document.addEventListener('scroll', this.handleParallaxScroll);

  },
  beforeDestroy: function() {
    document.removeEventListener('scroll', this.handleParallaxScroll);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleParallaxScroll: function() {
      let parallaxCity = document.querySelector('[purpose="parallax-city"]');
      let elementBottom = parallaxCity.offsetTop + parallaxCity.clientHeight;
      let viewportBottom = window.scrollY + window.innerHeight;
      let percentageScrolled;
      if (parallaxCity.offsetTop < viewportBottom && elementBottom > window.scrollY) {
        let visibleHeight = Math.min(elementBottom, viewportBottom) - Math.max(parallaxCity.offsetTop, window.scrollY);
        percentageScrolled = (visibleHeight / parallaxCity.clientHeight).toFixed(2);
        if(viewportBottom > elementBottom){
          percentageScrolled = 1;
        }
      } else {
        percentageScrolled = 0;
      }
      parallaxCity.querySelectorAll('div.layer').forEach((layer) => {
        let scrollAmount = layer.getAttribute('scroll-amount');
        let movement = (percentageScrolled * scrollAmount).toFixed(2);
        let translateY = 'translate3d(0, -' + movement + 'px, 0)';
        layer.style.transform = translateY;
      });
    },
  }
});
