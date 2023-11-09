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
    // Add an event listener to the image in the "ClickOps or DevOps" section.
    window.addEventListener('scroll', ()=>{
      let animatedImage = document.querySelector('[purpose="platform-animated-image"]');
      // Get the bounding box of the image.
      let animatedImageBoundingBox = animatedImage.getBoundingClientRect();
      if (animatedImageBoundingBox.top >= 0 &&
          animatedImageBoundingBox.left >= 0 &&
          animatedImageBoundingBox.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
          animatedImageBoundingBox.right <= (window.innerWidth || document.documentElement.clientWidth))
      {
        // When the image is completly in the user's viewport, add the 'animate' class to it.
        animatedImage.classList.add('animate');
      } else if(animatedImage.classList.contains('animate')) {
        // When it is no longer in the user's viewport, remove the 'animate' class if it has it.
        animatedImage.classList.remove('animate');
      }
    });
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
    }
  }
});
