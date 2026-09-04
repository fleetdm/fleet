parasails.registerPage('articles', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedArticles: [],
    filter: 'all',
    isArticlesLandingPage: false,
    articleCategory: '',
    categoryDescription: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {

    // Using the category to  articles,
    switch(this.category) {
      // If a specific category was provided, we'll set the articleCategory and categoryDescription.
      case 'securing':
        this.articleCategory = 'Security';
        this.categoryDescription = 'Learn more about how we secure Fleet.';
        break;
      case 'releases':
        this.articleCategory = 'Releases';
        this.categoryDescription = 'Read about the latest release of Fleet.';
        break;
      case 'engineering':
        this.articleCategory = 'Engineering';
        this.categoryDescription = 'Read about engineering at Fleet and beyond.';
        break;
      case 'guides':
        this.articleCategory = 'Guides';
        this.categoryDescription = 'Learn more about how to use Fleet to accomplish your goals.';
        break;
      case 'announcements':
        this.articleCategory = 'News';
        this.categoryDescription = 'The latest announcements from Fleet.';
        break;
      case 'podcasts':
        this.articleCategory = 'Podcasts';
        this.categoryDescription = 'Listen to the Future of Device Management podcast';
        break;
      case 'report':
        this.articleCategory = 'Reports';
        this.categoryDescription = '';
        break;
      case 'whitepapers':
        this.articleCategory = 'Whitepapers';
        this.categoryDescription = 'Browse our whitepapers to learn how modern teams manage and secure their devices.';
        break;
      case 'webinars':
        this.articleCategory = 'Webinars';
        this.categoryDescription = 'Watch Fleet and industry practitioners discuss real-world device management and IT operations.';
        break;
      case 'newsletters':
        this.articleCategory = 'Newsletters';
        this.categoryDescription = 'Catch up on past issues of the Fleet newsletter.';
        break;
      case 'industry-news':
        this.articleCategory = 'Industry news';
        this.categoryDescription = 'Device management and security news, and what it means for the devices you manage.';
        break;
      case 'articles':
        this.articleCategory = 'Blog';
        this.categoryDescription = 'Read the latest articles from the Fleet team and community.';
        break;
    }
  },

  mounted: async function() {
    if(['Blog', 'News', 'Guides', 'Releases'].includes(this.articleCategory)) {
      if(this.algoliaPublicKey) {// Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the handbook search will be disabled.
        docsearch({
          appId: 'NZXAYZXDGH',
          apiKey: this.algoliaPublicKey,
          indexName: 'fleetdm',
          container: '#docsearch-query',
          placeholder: 'Search articles',
          debug: false,
          clickAnalytics: true,
          searchParameters: {
            facetFilters: ['section:articles']
          },
          translations: {
            button: {
              buttonText: 'Search articles',
              buttonAriaLabel: 'Search articles',
            },
          },
        });
      }
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickGotoStart: function() {
      this.goto('/register');
    },
  }
});
