parasails.registerPage('articles', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedArticles: [],
    filter: 'all',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // this.selectedArticles = this.articles.sort((a, b)=>{
    //   if (a.meta['publishedOn'] > b.meta['publishedOn']) {
    //     return -1;
    //   }
    //   if ( b.meta['publishedOn'] < a.meta['publishedOn']){
    //     return 1;
    //   }
    // });
    this.selectedArticles = this.articles;
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    //…
    // sortArticlesByDate: function() {
    //   this.selectedArticles = this.articles.sort((a, b)=>{
    //     if (a.meta['publishedOn'] > b.meta['publishedOn']) {
    //       return -1;
    //     }
    //     if ( b.meta['publishedOn'] < a.meta['publishedOn']){
    //       return 1;
    //     }
    //   });
    // },

    // filterBy: function(filter) {
    //   if(filter !== 'all') {
    //     this.selectedArticles = this.articles.filter((article)=>{
    //       if(article.meta['category'] === filter) {
    //         return article;
    //       }
    //     });
    //   } else {
    //     this.sortArticlesByDate();
    //   }
    //   this.filter = filter;
    // }
  }
});
