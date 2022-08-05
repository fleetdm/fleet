parasails.registerPage('basic-handbook', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    isHandbookLandingPage: false,
    showHandbookNav: false,
    breadcrumbs: [],
    subtopics: [],
    handbookIndexLinks: [],

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
      appId: 'NZXAYZXDGH',
      apiKey: 'f3c02b646222734376a5e94408d6fead',
      indexName: 'fleetdm',
      inputSelector: '#docsearch-query',
      debug: false,
      clickAnalytics: true,
      algoliaOptions: {
        facetFilters: ['section:handbook']
      },
    });

    // Handle hashes in urls when coming from an external page.
    if(window.location.hash){
      let possibleHashToScrollTo = _.trimLeft(window.location.hash, '#');
      let hashToScrollTo = document.getElementById(possibleHashToScrollTo);
      // If the hash matches a header's ID, we'll scroll to that section.
      if(hashToScrollTo){
        hashToScrollTo.scrollIntoView();
      }
    }

    // If this is the handbook landing page, we'll generate the page links using the `linksForHandbookIndex` array that each handbook page has
    if(this.isHandbookLandingPage) {
      let handbookPages = [];
      for (let page of this.markdownPages) {
        if(_.startsWith(page.url, '/handbook') && !page.title.match(/^readme\.md$/i) && page.sectionRelativeRepoPath.match(/readme\.md$/i)) {
          let handbookPage = {
            pageTitle: page.title,
            url: page.url,
            pageLinks: page.linksForHandbookIndex,
          };
          handbookPages.push(handbookPage);
        }
      }
      // Sorting the handbook pages alphabetically by the pages url
      this.handbookIndexLinks = _.sortBy(handbookPages, 'url');
      // Sorting the company page to the top of the list, and the handbook page to the bottom
      this.handbookIndexLinks.sort((a)=>{
        if(a.pageTitle === '🔭 Company') {
          return -1;
        } else {
          return 0;
        }
      });
    }

    this.subtopics = (() => {
      let subtopics;
      if(!this.isHandbookLandingPage){
        subtopics = $('#body-content').find('h2.markdown-heading').map((_, el) => el.innerText);
      } else {
        subtopics = $('#body-content').find('h3').map((_, el) => el.innerText);
      }
      subtopics = $.makeArray(subtopics).map((title) => {
        // Removing all apostrophes from the title to keep  _.kebabCase() from turning words like 'user’s' into 'user-s'
        let kebabCaseFriendlyTitle = title.replace(/[\’]/g, '');
        return {
          title: title.replace(/([\uE000-\uF8FF]|\uD83C[\uDF00-\uDFFF]|\uD83D[\uDC00-\uDDFF])/g, ''), // take out any emojis (they look weird in the menu)
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
