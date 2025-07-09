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
      case 'success-stories':
        this.articleCategory = 'Success stories';
        this.categoryDescription = 'Read about how others are using Fleet and osquery.';
        break;
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
      case 'articles':
        this.articleCategory = 'Articles';
        this.categoryDescription = 'Read the latest articles from the Fleet team and community.';
        break;
    }
  },

  mounted: async function() {
    if(['Articles', 'Announcements', 'Guides', 'Releases'].includes(this.articleCategory)) {
      if(this.algoliaPublicKey) {// Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the handbook search will be disabled.
        docsearch({
          appId: 'NZXAYZXDGH',
          apiKey: this.algoliaPublicKey,
          indexName: 'fleetdm',
          container: '#docsearch-query',
          placeholder: 'Search',
          debug: false,
          clickAnalytics: true,
          searchParameters: {
            facetFilters: ['section:articles']
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
