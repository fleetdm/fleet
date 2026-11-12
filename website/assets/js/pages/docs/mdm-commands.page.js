parasails.registerPage('mdm-commands-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: 'apple',
    modal: undefined,
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
    if(window.location.hash) {
      if(window.location.hash === '#linux') {
        this.selectedPlatform = 'linux';
      } else if(window.location.hash === '#apple') {
        this.selectedPlatform = 'apple';
      } else if(window.location.hash === '#windows') {
        this.selectedPlatform = 'windows';
      }
      window.location.hash = '';
    }

    this.handleScrollingPlatformFilters();
    window.addEventListener('scroll', this.handleScrollingPlatformFilters);
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickSelectPlatform: function(platform) {
      this.selectedPlatform = platform;
      window.scrollTo({
        top: 0,
        left: 0,
        behavior: 'smooth',
      });
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
    clickOpenTableOfContents: function () {
      this.modal = 'table-of-contents';
    },
    closeModal: async function() {
      this.modal = '';
      await this.forceRender();
    }
  }
});
