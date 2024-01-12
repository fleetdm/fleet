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
      distanceFromBottomOfPage: undefined,
      isAnimating: false,
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
      this.distanceFromBottomOfPage = document.body.scrollHeight - this.distanceFromTopOfPage - (this.elementHeight * .5);
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
      let parallaxCityElementPosition = this.parallaxCityElement.getBoundingClientRect();
      if(parallaxCityElementPosition.bottom > this.distanceFromTopOfPage) {
        this.handleParallaxScroll();
      }

      this.parallaxCityElement.querySelectorAll('div').forEach((layer)=>{
        let initialPosition = layer.getAttribute('scroll-amount');
        layer.style.bottom = `-${Number(initialPosition) + 4}px`;
      });

      window.addEventListener('scroll', this.onScroll);
      window.addEventListener('resize', this.updateElementPositions);
      window.addEventListener('orientationchange', this.updateElementPositions);
    }
  },
  beforeDestroy: function() {
    if(!this.isMobile){
      window.removeEventListener('scroll', this.onScroll);
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
    onScroll: function() {
      if(!this.isAnimating){
        this.isAnimating = true;
        window.requestAnimationFrame(this.handleParallaxScroll);
      }
      return;
    },
    handleParallaxScroll: function() {
      let viewportBottom = window.scrollY + window.innerHeight;
      let percentageScrolled;
      if (this.parallaxCityElement.offsetTop < viewportBottom) {
        let visibleHeight = viewportBottom - Math.max(this.distanceFromTopOfPage, window.scrollY);
        percentageScrolled = visibleHeight / (this.distanceFromBottomOfPage + (this.elementHeight / 2 ));
      } else {
        percentageScrolled = 0;
      }
      if(percentageScrolled > 1){
        percentageScrolled = 1;
      }
      percentageScrolled = percentageScrolled.toFixed(4);
      if(percentageScrolled > .25){// When the element has been scrolled down 25%, start adjusting the position of layers.
        let adjustedPercentage = (percentageScrolled - .25) * 4/3;
        this.parallaxCityElement.querySelectorAll('div').forEach((layer) => {
          let scrollAmount = layer.getAttribute('scroll-amount');
          let movement = adjustedPercentage * scrollAmount;
          layer.style.transform = 'translateY(-' + movement + 'px)';
        });
      }
      this.isAnimating = false;
    },
  }
});
