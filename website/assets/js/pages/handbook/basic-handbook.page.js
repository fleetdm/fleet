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
    hideEmojisOnPage: false,
    regexToMatchEmoji: /(?:[\u00A9\u00AE\u203C\u2049\u2122\u2139\u2194-\u21AA\u2300-\u23FF\u2460-\u24FF\u25AA-\u25FE\u2600-\u26FF\u2700-\u27BF\u2900-\u297F\u2934-\u2935\u2B05-\u2B07\u2B1B-\u2B1C\u2B50\u2B55\u3030\u303D\u3297\u3299]|\uD83C[\uDC04\uDCCF\uDD70-\uDD71\uDD7E-\uDD7F\uDE00-\uDE02\uDE1A\uDE2F\uDE30-\uDE39\uDE3A-\uDE3F\uDE50-\uDE51\uDF00-\uDF21\uDF24-\uDF93\uDF96-\uDF97\uDF99-\uDFF0\uDFF3-\uDFF5\uDFF7-\uDFFF]|\uD83D[\uDC00-\uDDFF\uDE00-\uDE4F\uDE80-\uDEFF\uDFE0-\uDFFF]|\uD83E[\uDD0D-\uDDFF\uDE00-\uDEFF])\s{0,1}/g
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
    // If the user is on a windows device, hide emojis in the handbook.
    if(typeof bowser !== 'undefined' && bowser.windows) {
      this.hideEmojisOnPage = true;
      if(!this.isHandbookLandingPage){
        this.thisPage.title = this.thisPage.title.replace(this.regexToMatchEmoji, '');
        this._removeEmojiFromThisPage();
      }
    }
    // Adding a scroll event listener for scrolling sidebars and showing the back to top button.
    window.addEventListener('scroll', this.handleScrollingInHandbook);

    // Algolia DocSearch
    if(this.algoliaPublicKey) {// Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the handbook search will be disabled.
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: '#docsearch-query',
        placeholder: 'Search the handbook...',
        debug: false,
        clickAnalytics: true,
        searchParameters: {
          facetFilters: ['section:handbook']
        },
      });
    }

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
        if(
          _.startsWith(page.url, '/handbook/company')// Only add links for pages in the handbook/company/ folder
          && page.url !== '/handbook/company/handbook'// Hide the /handbook/company/handbook page in the handbook index.
          && !_.startsWith(page.url, '/handbook/company/open-positions')// Don't create links to pages generated for open positions.
        ) {
          let pageTitle = page.title;
          if(this.hideEmojisOnPage){
            pageTitle = pageTitle.replace(this.regexToMatchEmoji, '');
          }
          let handbookPage = {
            pageTitle: pageTitle,
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
        if(_.endsWith(a.pageTitle, 'Company')) {
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
        let kebabCaseFriendlyTitle = title.replace(/[\’\']/g, '');
        return {
          title: title.replace(this.regexToMatchEmoji, ''), // take out any emojis (they look weird in the menu)
          url: '#' + _.kebabCase(kebabCaseFriendlyTitle.toLowerCase()),
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
    handleScrollingInHandbook: function () {
      let backToTopButton = document.querySelector('div[purpose="back-to-top-button"]');
      let scrollTop = window.pageYOffset;
      if (backToTopButton) {
        if(scrollTop > 2500) {
          backToTopButton.classList.add('show');
        } else if (scrollTop === 0) {
          backToTopButton.classList.remove('show');
        }
      }

      this.scrollDistance = scrollTop;
    },
    _removeEmojiFromThisPage: function() {
      $('#body-content').html(
        $('#body-content').html()
        .replace(/✅/g, '&#x2713;')// Replace green checkmarks with unicode checkmarks
        .replace(/❌/g, '&#x2717;')// Replace red crosses with unicode crosses
        .replace(this.regexToMatchEmoji, '')// Remove all other emoji
      );
    },
    clickScrollToTop: function() {
      window.scrollTo({
        top: 0,
        left: 0,
        behavior: 'smooth',
      });
    }
  }
});
