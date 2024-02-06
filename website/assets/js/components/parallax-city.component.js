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
      parallaxCityElement: undefined,// For storing a jquery reference to the paralax-city-container div.
      parallaxLayers: [],// Stores an array of dictionaries, each containing a reference to a parallax-layer element, and the scroll-amount attribute
      elementBottomPosition: undefined,// For keeping track of the bottom position of the parllax image.
      elementHeight: undefined,// For keeping track of how large the parallax image element's height
      distanceFromTopOfPage: undefined, // Used to check if the image is within the user's viewport.
      lastScrolled: undefined,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div purpose="parallax-city-container">
      <div class="parallax-layer" purpose="background-cloud-3" scroll-amount=12></div>
      <div class="parallax-layer" purpose="background-cloud-2" scroll-amount=28></div>
      <div class="parallax-layer" purpose="small-island-2" scroll-amount=20></div>
      <div class="parallax-layer" purpose="small-island-1" scroll-amount=40></div>
      <div class="parallax-layer" purpose="background-cloud-1" scroll-amount=40></div>
      <div class="parallax-layer" purpose="large-island" scroll-amount=60></div>
      <div class="parallax-layer" purpose="foreground-cloud-2" scroll-amount=100></div>
      <div class="parallax-layer" purpose="foreground-cloud-1" scroll-amount=120></div>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

  },
  mounted: async function(){
    if(!bowser.isMobile){
      // Store a reference to the parent container, we'll use this to determine the elements position relative to the user's viewport.
      this.parallaxCityElement = $('div[purpose="parallax-city-container"]')[0];
      // Determine the parallax image's position on the page/user's viewport.
      this.getElementPositions();
      // If the bottom of the element is within the user's viewport, update the positions of the layers.
      if(this.parallaxCityElement.getBoundingClientRect().bottom > this.parallaxCityElement.offsetTop) {
        this.scrollParallaxLayers();
      }
      // Add a scroll event listener
      $(window).scroll(this.scrollParallaxLayers);
      // Add a resize event listener.
      $(window).resize(this.getElementPositions);
    }
  },
  beforeDestroy: function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    getElementPositions: function() {
      this.elementHeight = this.parallaxCityElement.clientHeight;
      this.distanceFromTopOfPage = this.parallaxCityElement.offsetTop;
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
    },
    scrollParallaxLayers: function() {
      // Calculate how much of the parallax image is visible.
      let visibleHeight = Math.min(this.elementBottomPosition, (window.scrollY + window.innerHeight)) - Math.max(this.distanceFromTopOfPage, window.scrollY);
      let percentageScrolled = visibleHeight / (this.elementHeight);
      // When the element has been scrolled down 50%,add the 'scroll-up' class to trigger the animation.
      if(percentageScrolled > .50) {
        $('div[purpose="parallax-city-container"]').addClass('scroll-up');
        $('div[purpose="parallax-city-container"]').removeClass('scroll-down');
      } else if(window.scrollY < this.lastScrolled) {// If the page is scrolled upwards, reset the
        $('div[purpose="parallax-city-container"]').removeClass('scroll-up');
        $('div[purpose="parallax-city-container"]').addClass('scroll-down');
      }
      this.lastScrolled = window.scrollY;
    },
  }
});
