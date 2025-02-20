parasails.registerPage('basic-documentation', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
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
    lastScrollTop: 0,
    modal: undefined,
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

    this.breadcrumbs = _.trim(this.thisPage.url, /\//).split(/\//);

    this.pages = _.sortBy(this.markdownPages, 'htmlId');

    this.pages = this.pages.filter((page)=>{
      return _.startsWith(page.url, '/docs');
    });
    this.pagesBySectionSlug = (() => {
      const DOCS_SLUGS = ['get-started', 'deploy', 'configuration', 'rest-api'];
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
    window.addEventListener('scroll', this.handleScrollingInDocumentation);
  },

  mounted: async function() {

    // Set a currentDocsSection value to display different Fleet premium CTAs based on what section is being viewed.
    this.currentDocsSection = this.thisPage.url.split(/\//).slice(-2)[0];

    // Handle hashes in urls when coming from an external page.
    if(window.location.hash){
      // If a hash was provided, we'll remove the # and any query parameters from it. (e.g., #create-an-api-only-user?utm_medium=fleetui&utm_campaign=get-api-token » create-an-api-only-user)
      // Note: Hash links for headings in markdown content will never have a '?' beacause they are removed when convereted to kebab-case, so we can safely strip everything after one if a url contains a query parameter.
      let possibleHashToScrollTo = _.trimLeft(window.location.hash.split('?')[0], '#');
      let elementWithMatchingId = document.getElementById(possibleHashToScrollTo);
      // If the hash matches a header's ID, we'll scroll to that section.
      if(elementWithMatchingId){
        // Get the distance of the specified element, and reduce it by 90 so the section is not hidden by the page header.
        let amountToScroll = elementWithMatchingId.offsetTop - 90;
        window.scrollTo({
          top: amountToScroll,
          left: 0,
        });
      }
    }

    // // Alternative jQuery approach to grab `on this page` links from top of markdown files
    // let subtopics = $('#body-content').find('h1 + ul').children().map((_, el) => el.innerHTML);
    // subtopics = $.makeArray(subtopics);
    // console.log(subtopics);

    this.subtopics = (() => {
      let subtopics = $('#body-content').find('h2.markdown-heading').map((_, el) => el);
      subtopics = $.makeArray(subtopics).map((subheading) => {
        return {
          title: subheading.innerText,
          url: $(subheading).find('a.markdown-link').attr('href'),
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

    // Set counters for items in ordered lists to be the value of their "start" attribute.
    document.querySelectorAll('ol[start]').forEach((ol)=> {
      let startValue = parseInt(ol.getAttribute('start'), 10) - 1;
      ol.style.counterReset = 'custom-counter ' + startValue;
    });

    // Adding event handlers to the links nested in headings on the page, allowing users to copy links by clicking on the link icon next to the heading.
    let headingsOnThisPage = $('#body-content').find(':header');
    for(let key in Object.values(headingsOnThisPage)){
      let heading = headingsOnThisPage[key];
      // Find the child <a> element
      let linkElementNestedInThisHeading = _.first($(heading).find('a.markdown-link'));
      $(linkElementNestedInThisHeading).click(()=> {
        if(typeof navigator.clipboard !== 'undefined') {
          // If this heading has already been clicked and still has the copied class we'll just ignore this click
          if(!$(heading).hasClass('copied')){
            // If the link's href is missing, we'll copy the current url (and remove any hashes) to the clipboard instead
            if(linkElementNestedInThisHeading.href) {
              navigator.clipboard.writeText(linkElementNestedInThisHeading.href);
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

    clickSwagRequestCTA: function () {
      if(window.gtag !== undefined){
        gtag('event','fleet_website__swag_request');
      }
      if(window.lintrk !== undefined) {
        window.lintrk('track', { conversion_id: 18587105 });// eslint-disable-line camelcase
      }
      if(window.analytics !== undefined) {
        analytics.track('fleet_website__swag_request');
      }
      this.goto('https://kqphpqst851.typeform.com/to/ZfA3sOu0#from_page=docs');
    },

    clickCTA: function (slug) {
      this.goto(slug);
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

    // FUTURE: remove this function if we do not add subsections to docs sections.
    // findAndSortNavSectionsByUrl: function (url='') {
    //   let NAV_SECTION_ORDER_BY_DOCS_SLUG = {
    //     'using-fleet':['The basics', 'Device management', 'Vuln management', 'Security compliance', 'Osquery management', 'Dig deeper'],
    //   };
    //   let slug = _.last(url.split(/\//));
    //   //
    //   if(NAV_SECTION_ORDER_BY_DOCS_SLUG[slug]) {
    //     let orderForThisSection = NAV_SECTION_ORDER_BY_DOCS_SLUG[slug];
    //     let sortedSection = {};
    //     orderForThisSection.map((section)=>{
    //       sortedSection[section] = this.navSectionsByDocsSectionSlug[slug][section];
    //     });
    //     this.navSectionsByDocsSectionSlug[slug] = sortedSection;
    //   }
    //   return this.navSectionsByDocsSectionSlug[slug];
    // },

    getActiveSubtopicClass: function (currentLocation, url) {
      return _.last(currentLocation.split(/#/)) === _.last(url.split(/#/)) ? 'active' : '';
    },

    getTitleFromUrl: function (url) {
      return _
        .chain(url.split(/\//))
        .last()
        .split(/-/)
        .map((str) => str === 'fleet' ? 'Fleet' : str === 'rest' ? 'REST' : str === 'api' ? 'API' : str)
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
          rightNavBar.classList.add('header-hidden');
          this.lastScrollTop = scrollTop;
        } else if(scrollTop < this.lastScrollTop - 60) {
          rightNavBar.classList.remove('header-hidden');
          this.lastScrollTop = scrollTop;
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
    },
    clickOpenMobileSubtopicsNav: function() {
      this.modal = 'subtopics';
    },
    clickOpenMobileDocsNav: function() {
      this.modal = 'table-of-contents';
    },
    closeModal: async function() {
      this.modal = '';
      await this.forceRender();
    }
  }

});
