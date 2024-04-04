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
      distanceFromBottomOfPage: undefined, // Used to track the amount of distance between the bottom of the image, and the bottom of the page.
      parallaxLayersAreCurrentlyAnimating: false,
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div purpose="parallax-city-container">
      <div class="parallax-layer" purpose="background-cloud-2" scroll-amount=4></div>
      <div class="parallax-layer" purpose="background-cloud-1" scroll-amount=6></div>
      <div class="parallax-layer" purpose="small-island-2" scroll-amount=16></div>
      <div class="parallax-layer" purpose="small-island-1" scroll-amount=12></div>
      <div class="parallax-layer" purpose="large-island" scroll-amount=24></div>
      <div class="parallax-layer" purpose="foreground-cloud-2" scroll-amount=32></div>
      <div class="parallax-layer" purpose="foreground-cloud-1" scroll-amount=40></div>
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
      // Build an array of parallax layers, and set the initial bottom position of each layer to be negative the layer's scroll amount.
      for(let layer of $('div.parallax-layer')) {
        let scrollAmount = Number($(layer).attr('scroll-amount'));
        $(layer).css('bottom', `-${scrollAmount}px`);
        this.parallaxLayers.push({element: layer, scrollAmount});
      }
      // Determine the parallax image's position on the page/user's viewport.
      this.getElementPositions();
      // If the bottom of the element is within the user's viewport, update the positions of the layers.
      if(this.parallaxCityElement.getBoundingClientRect().bottom > this.parallaxCityElement.offsetTop) {
        this.scrollParallaxLayers();
      }
      // Add a scroll event listener
      $(window).scroll(this.throttleParallaxScroll);
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
      this.distanceFromBottomOfPage = document.body.scrollHeight - this.distanceFromTopOfPage - (this.elementHeight * .5);
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
    },
    scrollParallaxLayers: function() {
      if(!this.parallaxLayersAreCurrentlyAnimating) {
        this.parallaxLayersAreCurrentlyAnimating = true;
        // Calculate how much of the parallax image is visible.
        let visibleHeight = (window.scrollY + window.innerHeight) - Math.max(this.distanceFromTopOfPage, window.scrollY);
        let percentageScrolled = visibleHeight / (this.distanceFromBottomOfPage + (this.elementHeight / 2 ));
        // When the element has been scrolled down 25%, iterate through the layers and update their positions.
        if(percentageScrolled > .25 ){
          let adjustedPercentage = (percentageScrolled - .25) * 4/3;
          for(let layer of this.parallaxLayers) {
            let movement = Math.min(adjustedPercentage * layer.scrollAmount, layer.scrollAmount);
            // Update the position of each layer.
            $(layer.element).css('transform', 'translate3D(0, -' + movement + 'px, 0)');
          }
        }
      }
    },
    throttleParallaxScroll: function() {
      this.scrollParallaxLayers();
      setTimeout(()=>{
        this.parallaxLayersAreCurrentlyAnimating = false;
      }, 100);
    }
  }
});
