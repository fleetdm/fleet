parasails.registerPage('homepage', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    modal: undefined,
    selectedCategory: 'endpoint-ops'
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function(){

    let imageToAnimate = document.querySelector('[purpose="platform-animated-image"]');
    // Make sure the animation event listener never runs if the image is removed.
    if(imageToAnimate) {
      this._addEventListenerForAnimation(imageToAnimate);
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
      window.addEventListener('scroll', ()=>{
        // Get the bounding box of the image.
        let animatedImageBoundingBox = imageToAnimate.getBoundingClientRect();
        if (animatedImageBoundingBox.top >= 0 &&
            animatedImageBoundingBox.left >= 0 &&
            animatedImageBoundingBox.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
            animatedImageBoundingBox.right <= (window.innerWidth || document.documentElement.clientWidth))
        {
          // When the image is completly in the user's viewport, add the 'animate' class to it.
          imageToAnimate.classList.add('animate');
        } else if(imageToAnimate.classList.contains('animate')) {
          // When it is no longer in the user's viewport, remove the 'animate' class if it has it.
          imageToAnimate.classList.remove('animate');
        }
      });
    }
  }
});
