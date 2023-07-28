parasails.registerPage('basic-documentation', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {

    isDocsLandingPage: false,

    inputTextValue: '',
    inputTimers: {},
    searchString: '',
    showDocsNav: false,
    currentDocsSection: '',
    breadcrumbs: [],
    pages: [],
    pagesBySectionSlug: {},
    subtopics: [],
    relatedTopics: [],
    scrollDistance: 0,
    navSectionsByDocsSectionSlug: {},

  },

  computed: {
    currentLocation: function () {
      return window.location.href;
    }
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (this.thisPage.url === '/docs') {
      this.isDocsLandingPage = true;
    }

    this.breadcrumbs = _.trim(this.thisPage.url, /\//).split(/\//);

    this.pages = _.sortBy(this.markdownPages, 'htmlId');

    this.pages = this.pages.filter((page)=>{
      return _.startsWith(page.url, '/docs');
    });
    this.pagesBySectionSlug = (() => {
      const DOCS_SLUGS = ['get-started', 'deploy', 'using-fleet', 'configuration', 'rest-api'];
      let sectionSlugs = _.uniq(this.pages.map((page) => page.url.split(/\//).slice(-2)[0]));
      let pagesBySectionSlug = {};

      for (let sectionSlug of sectionSlugs) {
        pagesBySectionSlug[sectionSlug] = this.pages.filter((page) => {
          return sectionSlug === page.url.split(/\//).slice(-2)[0];
        });

        // Sorting pages by pageOrderInSectionPath value, README files do not have a pageOrderInSectionPath, and FAQ pages are added to the end of the sorted array below.
        pagesBySectionSlug[sectionSlug] = _.sortBy(pagesBySectionSlug[sectionSlug], (page) => {
          if (!page.sectionRelativeRepoPath.match(/README\.md$/i) && !page.sectionRelativeRepoPath.match(/FAQ\.md$/i)) {
            return page.pageOrderInSectionPath;
          }
        });
        this.navSectionsByDocsSectionSlug[sectionSlug] = _.groupBy(pagesBySectionSlug[sectionSlug], 'docNavCategory');
      }
      // We need to re-sort the top-level sections because their htmlIds do not reflect the correct order
      pagesBySectionSlug['docs'] = DOCS_SLUGS.map((slug) => {
        return pagesBySectionSlug['docs'].find((page) => slug === _.kebabCase(page.title));
      });
      // We need to move any FAQs to the end of its array
      for (let slug of DOCS_SLUGS) {
        let pages = pagesBySectionSlug[slug];
        let index = pages.findIndex((page) => page.title === 'FAQ');
        if (index === -1 || index === pages.length - 1) {
          break;
        } else {
          let removedPage = _.pullAt(pages, index);
          pages.push(...removedPage);
          pagesBySectionSlug[slug] = pages;
        }
      }

      return pagesBySectionSlug;
    })();
    // Adding a scroll event listener for scrolling sidebars and showing the back to top button.
    if(!this.isDocsLandingPage){
      window.addEventListener('scroll', this.handleScrollingInDocumentation);
    }
  },

  mounted: async function() {

    // Set a currentDocsSection value to display different Fleet premium CTAs based on what section is being viewed.
    if(!this.isDocsLandingPage){
      this.currentDocsSection = this.thisPage.url.split(/\//).slice(-2)[0];
    }

    // Algolia DocSearch
    if(this.algoliaPublicKey) { // Note: Docsearch will only be enabled if sails.config.custom.algoliaPublicKey is set. If the value is undefined, the documentation search will be disabled.
      docsearch({
        appId: 'NZXAYZXDGH',
        apiKey: this.algoliaPublicKey,
        indexName: 'fleetdm',
        container: (this.isDocsLandingPage ? '#docsearch-query-landing' : '#docsearch-query'),
        clickAnalytics: true,
        searchParameters: {
          'facetFilters': ['section:docs']
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

    // // Alternative jQuery approach to grab `on this page` links from top of markdown files
    // let subtopics = $('#body-content').find('h1 + ul').children().map((_, el) => el.innerHTML);
    // subtopics = $.makeArray(subtopics);
    // console.log(subtopics);

    this.subtopics = (() => {
      let subtopics = $('#body-content').find('h2.markdown-heading').map((_, el) => el.innerText);
      subtopics = $.makeArray(subtopics).map((title) => {
        // Removing all apostrophes from the title to keep  _.kebabCase() from turning words like 'user’s' into 'user-s'
        let kebabCaseFriendlyTitle = title.replace(/[\’\']/g, '');
        return {
          title,
          url: '#' + _.kebabCase(kebabCaseFriendlyTitle.toLowerCase()),
        };
      });
      return subtopics;
    })();

    // https://github.com/sailshq/sailsjs.com/blob/7a74d4901dcc1e63080b502492b03fc971d3d3b2/assets/js/functions/sails-website-actions.js#L177-L239
    (function highlightThatSyntax(){
      $('pre code').each((i, block) => {
        window.hljs.highlightElement(block);
      });

      // Make sure the <pre> tags whose code isn't being highlighted
      // has that nice muted look we like.
      $('.nohighlight').each(function() {
        var $codeBlock = $(this);
        $codeBlock.closest('pre').addClass('muted');
      });
      // Also make sure the 'usage' (and 'usage-*') code blocks have special styles.
      $('.usage,.usage-exec').each(function() {
        var $codeBlock = $(this);
        $codeBlock.closest('pre').addClass('usage-wrapper');
      });

      // Now let's make the `function` keywords blue like in sublime.
      $('.hljs-keyword').each(function() {
        var $highlightedKeyword = $(this);
        if($highlightedKeyword.text() === 'function') {
          $highlightedKeyword.removeClass('hljs-keyword');
          $highlightedKeyword.addClass('hljs-function-keyword');
        }
      });

      $('.hljs-built_in').each(function() {
        var $builtIn = $(this);
        var $parentCode = $builtIn.closest('code');
        var isJavascriptSyntax = $parentCode.hasClass('javascript');
        var isBashSyntax = $parentCode.hasClass('bash');
        // ...and make the `require()`s not yellow, also like in sublime.
        if(isJavascriptSyntax && $builtIn.text() === 'require') {
          $builtIn.removeClass('hljs-built_in');
        }
        // And don't highlight the word 'test' in the bash examples, e.g. for ('sails new test-project')
        if(isBashSyntax && $builtIn.text() === 'test') {
          $builtIn.removeClass('hljs-built_in');
        }
      });
    })();

    // Adding event handlers to the Headings on the page, allowing users to copy links by clicking on the heading.
    let headingsOnThisPage = $('#body-content').find(':header');
    for(let key in Object.values(headingsOnThisPage)){
      let heading = headingsOnThisPage[key];
      $(heading).click(()=> {
        if(typeof navigator.clipboard !== 'undefined') {
          // Find the child <a> element
          let linkToCopy = _.first($(heading).find('a.markdown-link'));
          // If this heading has already been clicked and still has the copied class we'll just ignore this click
          if(!$(heading).hasClass('copied')){
            // If the link's href is missing, we'll copy the current url (and remove any hashes) to the clipboard instead
            if(linkToCopy) {
              navigator.clipboard.writeText(linkToCopy.href);
            } else {
              navigator.clipboard.writeText(heading.baseURI.split('#')[0]);
            }
            // Add the copied class to the header to notify the user that the link has been copied.
            $(heading).addClass('copied');
            // Remove the copied class 5 seconds later, so we can notify the user again if they re-cick on this heading
            setTimeout(()=>{$(heading).removeClass('copied');}, 5000);
          }
        }
      });
    }

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickOpenChatWidget: function() {
      if(window.HubSpotConversations && window.HubSpotConversations.widget){
        window.HubSpotConversations.widget.open();
      }
    },
    clickCTA: function (slug) {
      window.location = slug;
    },

    isCurrentSection: function (section) {
      if (_.trim(this.thisPage.url, (/\//)).split(/\//).includes(_.last(_.trimRight(section.url, (/\//)).split(/\//)))) {
        return true;
      }
      return false;
    },

    findPagesByUrl: function (url='') {
      let slug;
      // if no url is passed, use the base url as the slug (e.g., 'docs' or 'handbook')
      if (!url) {
        slug = _.trim(this.thisPage.url, /\//).split(/\//)[0];
      } else {
        slug = _.last(url.split(/\//));
      }

      return this.pagesBySectionSlug[slug];
    },

    findAndSortNavSectionsByUrl: function (url='') {
      let NAV_SECTION_ORDER_BY_DOCS_SLUG = {
        'using-fleet':['The basics', 'Device management', 'Vuln management', 'Security compliance', 'Osquery management', 'Dig deeper'],
        'deploy':['Uncategorized','TBD','Deployment guides'],
      };
      let slug = _.last(url.split(/\//));
      //
      if(NAV_SECTION_ORDER_BY_DOCS_SLUG[slug]) {
        let orderForThisSection = NAV_SECTION_ORDER_BY_DOCS_SLUG[slug];
        let sortedSection = {};
        orderForThisSection.map((section)=>{
          sortedSection[section] = this.navSectionsByDocsSectionSlug[slug][section];
        });
        this.navSectionsByDocsSectionSlug[slug] = sortedSection;
      }
      return this.navSectionsByDocsSectionSlug[slug];
    },

    getActiveSubtopicClass: function (currentLocation, url) {
      return _.last(currentLocation.split(/#/)) === _.last(url.split(/#/)) ? 'active' : '';
    },

    getTitleFromUrl: function (url) {
      return _
        .chain(url.split(/\//))
        .last()
        .split(/-/)
        .map((str) => str === 'fleet' ? 'Fleet' : str)
        .join(' ')
        .capitalize()
        .value();
    },

    toggleDocsNav: function () {
      this.showDocsNav = !this.showDocsNav;
    },

    delayInput: function (callback, ms, label) {
      let inputTimers = this.inputTimers;
      return function () {
        label = label || 'defaultTimer';
        _.has(inputTimers, label) ? clearTimeout(inputTimers[label]) : 0;
        inputTimers[label] = setTimeout(callback, ms);
      };
    },

    setSearchString: function () {
      this.searchString = this.inputTextValue;
    },

    handleScrollingInDocumentation: function () {
      let rightNavBar = document.querySelector('div[purpose="right-sidebar"]');
      let backToTopButton = document.querySelector('div[purpose="back-to-top-button"]');
      let scrollTop = window.pageYOffset;
      let windowHeight = window.innerHeight;
      // If the right nav bar exists, add and remove a class based on the current scroll position.
      if (rightNavBar) {
        if (scrollTop > this.scrollDistance && scrollTop > windowHeight * 1.5) {
          rightNavBar.classList.add('header-hidden', 'scrolled');
        } else if (scrollTop === 0) {
          rightNavBar.classList.remove('header-hidden', 'scrolled');
        } else {
          rightNavBar.classList.remove('header-hidden');
        }
      }
      // If back to top button exists, add and remove a class based on the current scroll position.
      if (backToTopButton){
        if (scrollTop > 2500) {
          backToTopButton.classList.add('show');
        } else if (scrollTop === 0) {
          backToTopButton.classList.remove('show');
        }
      }
      this.scrollDistance = scrollTop;
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
