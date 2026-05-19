parasails.registerPage('device-management-glossary-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    searchQuery: '',
    // Indexed copy of the server-rendered terms, populated in beforeMount from
    // window.SAILS_LOCALS so search/filter can run without re-reading the DOM.
    termIndex: [],
    visibleTermCount: 0,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if (window.SAILS_LOCALS && _.isArray(window.SAILS_LOCALS.glossaryTerms)) {
      this.termIndex = window.SAILS_LOCALS.glossaryTerms.map((term) => {
        return {
          slug: term.slug,
          name: term.name,
          nameLower: term.name.toLowerCase(),
        };
      });
      this.visibleTermCount = this.termIndex.length;
    }
  },

  mounted: function() {
    // Honor a "?q=" param on initial load for shareable filtered views.
    let params = new URLSearchParams(window.location.search);
    let initialQuery = params.get('q');
    if (initialQuery) {
      this.searchQuery = initialQuery;
    }
  },

  //  ╦ ╦╔═╗╔╦╗╔═╗╦ ╦╔═╗╦═╗╔═╗
  //  ║║║╠═╣ ║ ║  ╠═╣║╣ ╠╦╝╚═╗
  //  ╚╩╝╩ ╩ ╩ ╚═╝╩ ╩╚═╝╩╚═╚═╝
  watch: {
    searchQuery: function() {
      this.recomputeVisibleCount();
    },
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    // A term card is visible when its header (name) contains the search query
    // (case-insensitive substring). No category filter; that bar was removed.
    termIsVisible: function(slug) {
      let term = _.find(this.termIndex, { slug: slug });
      if (!term) {
        return true;
      }
      let q = (this.searchQuery || '').trim().toLowerCase();
      if (q && term.nameLower.indexOf(q) === -1) {
        return false;
      }
      return true;
    },
    letterIsVisible: function(letter) {
      let termsForLetter = _.filter(this.termIndex, (t) => t.name.charAt(0).toUpperCase() === letter);
      return _.some(termsForLetter, (t) => this.termIsVisible(t.slug));
    },
    resetFilters: function() {
      this.searchQuery = '';
    },
    // Triggered when the user presses Enter in the search field.
    // Scrolls to the first term whose header matches the current query.
    jumpToFirstHeaderMatch: function() {
      let q = (this.searchQuery || '').trim().toLowerCase();
      if (!q) {
        return;
      }
      let match = _.find(this.termIndex, (t) => t.nameLower.indexOf(q) !== -1);
      if (!match) {
        return;
      }
      let el = document.getElementById('term-' + match.slug);
      if (!el) {
        return;
      }
      // Update the URL hash so :target highlight applies and the link is shareable.
      // Use replaceState so repeated Enter presses don't stack history entries.
      if (window.history && window.history.replaceState) {
        window.history.replaceState(null, '', '#term-' + match.slug);
      } else {
        window.location.hash = 'term-' + match.slug;
      }
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    },
    recomputeVisibleCount: function() {
      this.visibleTermCount = _.filter(this.termIndex, (t) => this.termIsVisible(t.slug)).length;
    },
  }
});
