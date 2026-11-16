parasails.registerPage('app-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: 'macos',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    if(bowser.windows){
      this.selectedPlatform = 'windows';
    }
    // Add a scroll event listener to shift the platform filters upwards when the header is hidden.
    window.addEventListener('scroll', this.handleScrollingPlatformFilters);
    this.handleScrollingPlatformFilters();
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickSelectPlatform: function(platform) {
      // Scroll back up to the top of the list of apps (the feature cards above the platform filters push it down the page).
      let appListTop = document.querySelector('div[purpose="count-and-cta"]').getBoundingClientRect().top + window.scrollY - 200;
      if(this.selectedPlatform !== platform && window.scrollY > appListTop + 20){
        window.scrollTo(0, appListTop);
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

  }
});
