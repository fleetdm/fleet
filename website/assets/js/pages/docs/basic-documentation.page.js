parasails.registerPage('basic-documentation', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {

    isLandingPage: false,

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
    //…
  },

  mounted: async function() {
    this.breadcrumbs = _.trim(this.thisPage.url, '/').split('/');

    this.pages = _.sortBy(this.markdownPages, 'htmlId');

    let sections = _.uniq(_.pluck(this.pages, 'url').map((url) => url.split(/\//).slice(-2)[0]));

    for (let sectionSlug of sections) {
      this.pagesBySectionSlug[sectionSlug] = _
        .chain(this.pages)
        .filter((page) => {
          return sectionSlug === page.url.split(/\//).slice(-2)[0];
        })
        .sortBy((page) => {
          // custom sort function is needed because simple sort of alphanumeric htmlIds strings
          // does not appropriately handle double-digit strings
          try {
            // attempt to split htmlId and parse out its ordinal value (e.g., `docs--10-teams--xxxxxxxxxx`)
            let sortValue = page.htmlId.split('--')[1].split('-')[0];
            return parseInt(sortValue) || sortValue;
          } catch (error) {
            // something unexpected happened so just return the htmlId and continue sort
            console.log(error);
            return page.htmlId;
          }
        })
        .value();
    }
    console.log('pagesBySectionSlug: ', this.pagesBySectionSlug);

    // // Alternative jQuery approach to grab `on this page` links from top of markdown files
    // let subtopics = $('#body-content').find('h1 + ul').children().map((_, el) => el.innerHTML);
    // subtopics = $.makeArray(subtopics);
    // console.log(subtopics);

    let subtopicsList = $('#body-content').find('h2').map((_, el) => el.innerHTML);
    this.subtopics = $.makeArray(subtopicsList).map((title) => {
      return {
        title,
        url: '#' + _.kebabCase(title),
      };
    });


    // https://github.com/sailshq/sailsjs.com/blob/7a74d4901dcc1e63080b502492b03fc971d3d3b2/assets/js/functions/sails-website-actions.js#L177-L239
    (function highlightThatSyntax(){
      $('pre code').each(function(i, block) {
        hljs.highlightBlock(block);
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

    isCurrentSection: function (section) {
      if (_.trim(this.thisPage.url, ('/')).split('/').includes(_.last(_.trimRight(section.url, ('/')).split('/')))) {
        console.log('isCurrentSection: ', section);
        return true;
      }
      return false;
    },

    getActiveSubtopicClass: function (currentLocation, url) {
      return _.last(currentLocation.split('#')) === _.last(url.split('#')) ? 'active' : '';
    },

    findPageByUrl: function (url) {
      return this.pages.find((page) => page.url === url);
    },

    getPagesBySectionSlug: function (slug='') {
      if (!slug) {
        slug = _.trim(this.thisPage.url, '/').split('/')[0];
      }
      return this.pagesBySectionSlug[slug];
    },

    // TODO remove this after MM fixes titles for readmes in build script
    getTitle: function (page) {
      if (page.title && page.title === 'README') {
        return _.chain(page.url.split('/'))
          .last()
          .split('-')
          .map((str) => str === 'fleet' ? 'Fleet' : str)
          .join(' ')
          .capitalize()
          .value();
      }
      return page.title;
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
