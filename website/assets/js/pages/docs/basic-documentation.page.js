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

    breadcrumbs: [],
    pages: [],
    pagesBySectionSlug: {},
    subtopics: [],
    relatedTopics: [],

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

    this.pagesBySectionSlug = (() => {
      const DOCS_SLUGS = ['using-fleet', 'deploying', 'contributing'];

      let sectionSlugs = _.uniq(_.pluck(this.pages, 'url').map((url) => url.split(/\//).slice(-2)[0]));

      let pagesBySectionSlug = {};

      for (let sectionSlug of sectionSlugs) {
        pagesBySectionSlug[sectionSlug] = _
          .chain(this.pages)
          .filter((page) => {
            return sectionSlug === page.url.split(/\//).slice(-2)[0];
          })
          .sortBy((page) => {
            // custom sort function is needed because simple sort of alphanumeric htmlIds strings
            // does not appropriately handle double-digit strings
            try {
              // attempt to split htmlId and parse out its ordinal value (e.g., `docs--10-teams--xxxxxxxxxx`)
              let sortValue = page.htmlId.split(/--/)[1].split(/-/)[0];
              return parseInt(sortValue) || sortValue;
            } catch (error) {
              // something unexpected happened so just return the htmlId and continue sort
              console.log(error);
              return page.htmlId;
            }
          })
          .value();
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
  },

  mounted: async function() {

    // // Alternative jQuery approach to grab `on this page` links from top of markdown files
    // let subtopics = $('#body-content').find('h1 + ul').children().map((_, el) => el.innerHTML);
    // subtopics = $.makeArray(subtopics);
    // console.log(subtopics);

    this.subtopics = (() => {
      let subtopics = $('#body-content').find('h2').map((_, el) => el.innerHTML);
      subtopics = $.makeArray(subtopics).map((title) => {
        return {
          title,
          url: '#' + _.kebabCase(title),
        };
      });
      return subtopics;
    })();

    // https://github.com/sailshq/sailsjs.com/blob/7a74d4901dcc1e63080b502492b03fc971d3d3b2/assets/js/functions/sails-website-actions.js#L177-L239
    (function highlightThatSyntax(){
      $('pre code').each((i, block) => {
        window.hljs.highlightBlock(block);
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

  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

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

  }

});
