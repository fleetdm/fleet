
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

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },

  mounted: async function() {
    this.breadcrumbs = _.trim(this.thisPage.url, '/').split('/');
    console.log('breadcrumbs: ', this.breadcrumbs);

    this.pages = _.sortBy(this.markdownPages, 'htmlId');
    console.log('pages: ', this.pages);

    let sections = _.uniq(_.pluck(this.pages, 'url').map((url) => url.split(/\//).slice(-2)[0]));
    console.log('sections: ', sections);

    for (let sectionSlug of sections) {
      this.pagesBySectionSlug[sectionSlug] = _.sortBy(this.pages.filter((page) => {
        return sectionSlug === page.url.split(/\//).slice(-2)[0];
      }), 'htmlId');
    }
    console.log('pagesBySectionSlug: ', this.pagesBySectionSlug);

    let subtopics = $('#body-content').find('h1 + ul').children().map((_, el) => el.innerHTML);
    this.subtopics = $.makeArray(subtopics);

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

    findPageByUrl: function (url) {
      return this.pages.find((page) => page.url === url);
    },

    getPagesBySectionSlug: function (slug='') {
      if (!slug) {
        slug = _.trim(this.thisPage.url, '/').split('/')[0];
      }
      console.log(slug);

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
