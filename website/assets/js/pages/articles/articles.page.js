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

    if (this.category === 'all') {
      // if the category is set to 'all', we'll show the articles landing page and set `isArticlesLandingPage` to true
      this.isArticlesLandingPage = true;
    } else {
      switch(this.category) {
        // If a specific category was provided, we'll set the articleCategory and categoryDescription.
        case 'device-management':
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
          this.articleCategory = 'Announcements';
          this.categoryDescription = 'The latest news from Fleet.';
          break;
        case 'deploy':
          this.articleCategory = 'Deployment guides';
          this.categoryDescription = 'Learn more about how to deploy Fleet.';
          break;
        case 'podcasts':
          this.articleCategory = 'Podcasts';
          this.categoryDescription = 'Listen to the Future of Device Management podcast';
          break;
        case 'report':
          this.articleCategory = 'Reports';
          this.categoryDescription = '';
          break;
      }
    }
    // Sorting articles on the page based on their 'publishedOn' date.
    this.sortArticlesByDate();
  },

  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {


    sortArticlesByDate: function() {

      this.selectedArticles = this.articles.sort((a, b)=>{
        if (a.meta['publishedOn'] > b.meta['publishedOn']) {
          return -1;
        }
        if (b.meta['publishedOn'] > a.meta['publishedOn']){
          return 1;
        }
      });
    },

    filterBy: function(filter) {
      if(filter !== 'all') {
        this.selectedArticles = this.articles.filter((article)=>{
          if(article.meta['category'] === filter) {
            return article;
          }
        });
      } else {
        this.sortArticlesByDate();
      }
      this.filter = filter;
    }
  }
});
