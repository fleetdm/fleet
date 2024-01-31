parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: undefined,
    selectedCategory: 'device-management'
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
    // TODO: tear down  (it seems crazy, but do it for yourself, in case you need to move this later.  I know it won't actually matter)
  },
  mounted: async function(){

    let imageToAnimate = document.querySelector('[purpose="platform-animated-image"]');
    // Make sure the animation event listener never runs if the image is removed.  TODO: THis should never happen, thus throw an error if it ever happens.  IF it is known and expected, then handle that explicitly and have the code understand why.
    if(imageToAnimate) {
      // TODO: Where is mobile disabling happening?  (Let's disable it here instead with `if (bowser.isMobile)`)
      this._addEventListenerForAnimation(imageToAnimate);// «« TODO: This can be inlined (remove private helper function), tehre is no other place it is called.  If there are hidden sneaky places it is called from the html then we should not do that.
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {


    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },

    clickOpenVideoModal: function(modalName) {
      this.modal = modalName;
    },

    closeModal: function() {
      this.modal = undefined;
    },

    _addEventListenerForAnimation: function (imageToAnimate) {
      window.addEventListener('scroll', ()=>{// « TODO: Change this to use jQuery
        // Get the bounding box of the image.
        let animatedImageBoundingBox = imageToAnimate.getBoundingClientRect();
        if (animatedImageBoundingBox.top >= 0 &&
            animatedImageBoundingBox.left >= 0 &&
            animatedImageBoundingBox.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
            animatedImageBoundingBox.right <= (window.innerWidth || document.documentElement.clientWidth))
        {
          // When the image is completly in the user's viewport, add the 'animate' class to it.
          imageToAnimate.classList.add('animate');// « TODO: Change this to use jQuery.  (which automatically optimizes DOM renders so you don't potentially add the same class again and again and again, triggering fresh DOM renders each time)
          // TODO: ^where is this clsas implemented?  Can't find in CSS.  Can't find in Bootstrap.
          // TODO: IF it's vue.js animations, then lets stop.  They don't work very well.
        }
      });
    }
  }
});
