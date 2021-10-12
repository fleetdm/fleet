parasails.registerPage('basic-handbook', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    isHandbookLandingPage: false,
    showHandbookNav: false,
    breadcrumbs: [],
    subtopics: [],

  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (this.thisPage.url === '/handbook') {
      this.isHandbookLandingPage = true;
    }
    this.breadcrumbs = _.trim(this.thisPage.url, /\//).split(/\//);

  },

  mounted: async function() {

    // Algolia DocSearch
    docsearch({
      appId: 'X0HLT4WWY2',
      apiKey: '8caf918a6047be5b0897c20603d18d01',
      indexName: 'handbook',
      inputSelector: '#docsearch-query',
      debug: false,
      clickAnalytics: true,
    });
    // Algolia DocSearch
    docsearch({
      appId: 'X0HLT4WWY2',
      apiKey: '8caf918a6047be5b0897c20603d18d01',
      indexName: 'handbook',
      inputSelector: '#docsearch-query-mobile',
      debug: false,
      clickAnalytics: true,
    });

    this.subtopics = (() => {
      let subtopics;
      if(!this.isHandbookLandingPage){
        subtopics = $('#body-content').find('h2').map((_, el) => el.innerText);
      } else {
        subtopics = $('#body-content').find('h3').map((_, el) => el.innerText);
      }
      subtopics = $.makeArray(subtopics).map((title) => {
        // Removing all apostrophes from the title to keep  _.kebabCase() from turning words like 'user’s' into 'user-s'
        let kebabCaseFriendlyTitle = title.replace(/[\’]/g, '');
        return {
          title,
          url: '#' + _.kebabCase(kebabCaseFriendlyTitle),
        };
      });
      return subtopics;
    })();
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    _isCurrentSection: function (section, location) {
      if (location.hash === section.url) {
        return true;
      }
      return false;
    },
    _getTitleFromUrl: function (url) {
      return _
        .chain(url.split(/\//))
        .last()
        .split(/-/)
        .map((str) => str === 'fleet' ? 'Fleet' : str)
        .join(' ')
        .capitalize()
        .value();
    },
  }
});
