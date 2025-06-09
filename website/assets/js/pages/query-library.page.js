parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: 'macos', // Initially set to 'macos'
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function () {
    //…
  },
  mounted: async function () {
    if(bowser.windows){
      this.selectedPlatform = 'windows';
    }
    window.addEventListener('scroll', this.handleScrollingPlatformFilters);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickSelectPlatform: function(platform) {
      if(this.selectedPlatform !== platform && window.scrollY > 300){
        window.scrollTo(0, 300, {smooth: true});
      }
      this.selectedPlatform = platform;
    },

    handleScrollingPlatformFilters: function () {
      let platformFilters = document.querySelector('div[purpose="platform-filters"]');
      let scrollTop = window.pageYOffset;
      let windowHeight = window.innerHeight;
      // If the right nav bar exists, add and remove a class based on the current scroll position.
      if (platformFilters) {
        if (scrollTop > this.scrollDistance && scrollTop > windowHeight * 1.5) {
          platformFilters.classList.add('header-hidden');
          this.lastScrollTop = scrollTop;
        } else if(scrollTop < this.lastScrollTop - 60) {
          platformFilters.classList.remove('header-hidden');
          this.lastScrollTop = scrollTop;
        }
      }
      this.scrollDistance = scrollTop;
    },

  },

});
