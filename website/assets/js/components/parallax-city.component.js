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
  props: ['isMobile'],

  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: function (){
    return {
      parallaxCityElement: undefined,
      elementBottomPosition: undefined,
      elementHeight: undefined,
      distanceFromTopOfPage: undefined,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div purpose="parallax-city-container">
      <div purpose="background-cloud-3" scroll-amount=12></div>
      <div purpose="background-cloud-2" scroll-amount=28></div>
      <div purpose="small-island-2" scroll-amount=20></div>
      <div purpose="small-island-1" scroll-amount=40></div>
      <div purpose="background-cloud-1" scroll-amount=40></div>
      <div purpose="large-island" scroll-amount=60></div>
      <div purpose="swans" scroll-amount=60></div>
      <div purpose="foreground-cloud-2" scroll-amount=100></div>
      <div purpose="foreground-cloud-1" scroll-amount=120></div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function(){
    if(!this.isMobile){
      this.parallaxCityElement = document.querySelector('[purpose="parallax-city-container"]');
      this.elementHeight = this.parallaxCityElement.clientHeight;
      this.distanceFromTopOfPage = this.parallaxCityElement.offsetTop;
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
      let parallaxCityElementPosition = this.parallaxCityElement.getBoundingClientRect();
      if(parallaxCityElementPosition.bottom > this.distanceFromTopOfPage) {
        this.handleParallaxScroll();
      }

      this.parallaxCityElement.querySelectorAll('div').forEach((layer)=>{
        let initialPosition = layer.getAttribute('scroll-amount');
        layer.style.bottom = `-${Number(initialPosition) + 4}px`;
      });

      window.addEventListener('scroll', this.handleParallaxScroll);
      window.addEventListener('resize', this.updateElementPositions);
      window.addEventListener('orientationchange', this.updateElementPositions);
    }
  },
  beforeDestroy: function() {
    if(!this.isMobile){
      window.removeEventListener('scroll', this.handleParallaxScroll);
      window.removeEventListener('resize', this.updateElementPositions);
      window.removeEventListener('orientationchange', this.updateElementPositions);
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    updateElementPositions: function() {
      this.elementHeight = this.parallaxCityElement.clientHeight;
      this.distanceFromTopOfPage = this.parallaxCityElement.offsetTop;
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
    },
    handleParallaxScroll: function() {
      let viewportBottom = window.scrollY + window.innerHeight;
      let percentageScrolled;
      if (this.parallaxCityElement.offsetTop < viewportBottom && this.elementBottomPosition > window.scrollY) {
        let visibleHeight = Math.min(this.elementBottomPosition, viewportBottom) - Math.max(this.distanceFromTopOfPage, window.scrollY);
        percentageScrolled = (visibleHeight / this.elementHeight);
        if(viewportBottom > this.elementBottomPosition) { // If the page is scrolled past the element, set the percentage scrolled to 1.
          percentageScrolled = 1;
        }
      } else {
        percentageScrolled = 0;
      }
      if(percentageScrolled > 0){
        this.parallaxCityElement.querySelectorAll('div').forEach((layer) => {
          let scrollAmount = layer.getAttribute('scroll-amount');
          let movement = (percentageScrolled * scrollAmount);
          layer.style.transform = 'translateY(-' + movement + 'px)';
        });
      }
    },
  }
});
