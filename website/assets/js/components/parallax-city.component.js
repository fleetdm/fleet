/**
 * <parallax-city>
 * -----------------------------------------------------------------------------
 * An image of Fleet cloud city with a slight parallax scrolling effect.
 * or a static image for mobile devices and browsers with hardware acceleration disabled.
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
      enableAnimation: true,// Whether or not to disable the parallax scrolling animation.
    };
  },

  //  ╦ ╦╔╦╗╔╦╗╦
  //  ╠═╣ ║ ║║║║
  //  ╩ ╩ ╩ ╩ ╩╩═╝
  template: `
  <div>
    <div purpose="parallax-city-container" v-if="enableAnimation">
      <div class="parallax-layer" purpose="background-cloud-2" scroll-amount="4"></div>
      <div class="parallax-layer" purpose="background-cloud-1" scroll-amount="6"></div>
      <div class="parallax-layer" purpose="small-island-2" scroll-amount="16"></div>
      <div class="parallax-layer" purpose="small-island-1" scroll-amount="12"></div>
      <div class="parallax-layer" purpose="large-island" scroll-amount="24"></div>
      <div class="parallax-layer" purpose="foreground-cloud-2" scroll-amount="32"></div>
      <div class="parallax-layer" purpose="foreground-cloud-1" scroll-amount="40"></div>
    </div>
    <div purpose="static-cloud-city" v-else>
    </div>
  </div>
  `,

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // Disable animation on mobile devices.
    if(bowser.isMobile) {
      this.enableAnimation = false;
    }
    // Check for hardware/graphics acceleration.
    if(bowser.chrome || bowser.opera) {
      this.enableAnimation = this._isHardwareAccelerationEnabledOnChromiumBrowsers();
    } else if(bowser.firefox){
      this.enableAnimation = this._isHardwareAccelerationEnabledOnFirefox();
    }
  },
  mounted: async function(){
    if(this.enableAnimation) {
      this._setupParallaxAnimation();
    }
  },
  beforeDestroy: function() {

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    _getElementPositions: function() {
      this.elementHeight = this.parallaxCityElement.clientHeight;
      this.distanceFromTopOfPage = this.parallaxCityElement.offsetTop;
      this.distanceFromBottomOfPage = document.body.scrollHeight - this.distanceFromTopOfPage - (this.elementHeight * .5);
      this.elementBottomPosition = this.elementHeight + this.distanceFromTopOfPage;
    },
    _setupParallaxAnimation: function() {
      // Store a reference to the parent container, we'll use this to determine the elements position relative to the user's viewport.
      this.parallaxCityElement = $('div[purpose="parallax-city-container"]')[0];
      // Build an array of parallax layers, and set the initial bottom position of each layer to be negative the layer's scroll amount.
      for(let layer of $('div.parallax-layer')) {
        let scrollAmount = Number($(layer).attr('scroll-amount'));
        $(layer).css('bottom', `-${scrollAmount + 1}px`);
        this.parallaxLayers.push({element: layer, scrollAmount});
      }
      // Determine the parallax image's position on the page/user's viewport.
      this._getElementPositions();
      // If the bottom of the element is within the user's viewport, update the positions of the layers.
      if(this.parallaxCityElement.getBoundingClientRect().bottom > this.parallaxCityElement.offsetTop) {
        this.scrollParallaxLayers();
      }
      // Add a scroll event listener
      $(window).scroll(this._throttleParallaxScroll);
      // Add a resize event listener.
      $(window).resize(this._getElementPositions);
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
    _throttleParallaxScroll: function() {
      this.scrollParallaxLayers();
      setTimeout(()=>{
        this.parallaxLayersAreCurrentlyAnimating = false;
      }, 100);
    },
    _isHardwareAccelerationEnabledOnChromiumBrowsers: function() {
      let isHardwareAccelerationEnabled = true;
      // For Chromium based browsers, we'll check the vendor of the user's graphics card.
      // See https://gist.github.com/cvan/042b2448fcecefafbb6a91469484cdf8 for more info about this method.
      let canvas = document.createElement('canvas');
      let webGLContext = canvas.getContext('webgl');
      if (!webGLContext) {
        // If webGLContext is undefined, we'll assume the user has hardware acceleration disabled, and we won't animate the parallax layers.
        isHardwareAccelerationEnabled = false;
      } else {
        // Otherwise, we'll check to see if the 'Vendor' of this users GPU.
        let debugInfo = webGLContext.getExtension('WEBGL_debug_renderer_info');
        let vendor = webGLContext.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
        // If vendor is "Google Inc. (Google)" or "Google Inc.", we can safely assume this user doesn't have hardware acceleration enabled and we'll disable the parallax animation.
        if(vendor === 'Google Inc. (Google)' || vendor === 'Google Inc.') {
          isHardwareAccelerationEnabled = false;
        }
      }
      return isHardwareAccelerationEnabled;
    },
    _isHardwareAccelerationEnabledOnFirefox: function() {
      // For Firefox, the method we use for chrome does not always work.
      // Instead, we'll run two tests, one with forced software rendering, and one without to see if the results are the same.
      // See https://stackoverflow.com/a/77170999 for more info about this method.
      let canvas = document.createElement('canvas');
      let ctx = canvas.getContext('2d', { willReadFrequently: false });
      ctx.moveTo(0, 0);
      ctx.lineTo(120, 121);
      ctx.stroke();
      let firstTestResults = ctx.getImageData(0, 0, 200, 200).data.join();
      let canvasForSoftwareRenderingTest = document.createElement('canvas');
      let ctxWithSoftwareRendering = canvasForSoftwareRenderingTest.getContext('2d', { willReadFrequently: true });// willReadFrequently will force software rendering
      ctxWithSoftwareRendering.moveTo(0, 0);
      ctxWithSoftwareRendering.lineTo(120, 121); // HWA is bad at obliques
      ctxWithSoftwareRendering.stroke();
      let softwareRenderingTestResults = ctxWithSoftwareRendering.getImageData(0, 0, 200, 200).data.join();
      // If the results from the software rendering test are identical to the first test, we can assume the user has hardware acceleration disabled.
      return firstTestResults !== softwareRenderingTestResults;
    },
  }
});
