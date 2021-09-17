parasails.registerPage('basic-handbook', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    isHandbookLandingPage: false,

    inputTextValue: '',
    inputTimers: {},
    searchString: '',
    showHandbookNav: false,

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
    if (this.thisPage.url === '/handbook') {
      this.isHandbookLandingPage = true;
    }

    this.breadcrumbs = _.trim(this.thisPage.url, /\//).split(/\//);

    this.pages = _.sortBy(this.markdownPages, 'htmlId');

    this.pagesBySectionSlug = (() => {
      const HANDBOOK_SLUGS = ['handbook'];

      let sectionSlugs = _.uniq(_.pluck(this.pages, 'url').map((url) => url.split(/\//).slice(-2)[0]));
      console.log(sectionSlugs);
      let pagesBySectionSlug = {};
      for (let sectionSlug of sectionSlugs) {
        pagesBySectionSlug[sectionSlug] = _
          .chain(this.pages)
          .filter((page) => {
            console.log(page);
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
      return pagesBySectionSlug;
    })();
    console.log(this.pagesBySectionSlug);
  },
  mounted: async function() {
    //…
    this.subtopics = (() => {
      let subtopics = $('#body-content').find('h3').map((_, el) => el.innerText);
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
    toggleHandbookNav: function () {
      this.showHandbookNav = !this.showHandbookNav;
    },
    //…
  }
});
