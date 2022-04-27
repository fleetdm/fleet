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
    this.getCategoryInformation();
    this.sortArticlesByDate();
  },

  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    getCategoryInformation: function() {
      console.log(this.category);
      if (this.category === 'all') {
        this.isArticlesLandingPage = true;
      } else {
        switch(this.category) {
          case 'device-management':
            this.articleCategory = 'Success stories';
            this.categoryDescription = '';
            break;
          case 'securing':
            this.articleCategory = 'Security';
            this.categoryDescription = '';
            break;
          case 'releases':
            this.articleCategory = 'Releases';
            this.categoryDescription = '';
            break;
          case 'engineering':
            this.articleCategory = 'Engineering';
            this.categoryDescription = '';
            break;
          case 'guides':
            this.articleCategory = 'Guides';
            this.categoryDescription = 'Learn more about how to deploy and use Fleet.';
            break;
          case 'announcements':
            this.articleCategory = 'Announcements';
            this.categoryDescription = '';
            break;
          case 'use-cases':
            this.articleCategory = 'Product';
            this.categoryDescription = '';
            break;
        }
      }
    },

    sortArticlesByDate: function() {
      this.selectedArticles = this.articles.sort((a, b)=>{
        if (a.meta['publishedOn'] > b.meta['publishedOn']) {
          return -1;
        }
        if ( b.meta['publishedOn'] < a.meta['publishedOn']){
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
