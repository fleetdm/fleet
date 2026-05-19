parasails.registerPage('device-management-glossary-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    searchQuery: '',
    // Indexed copy of the server-rendered terms, populated in beforeMount from
    // window.SAILS_LOCALS so search/filter can run without re-reading the DOM.
    termIndex: [],
    termBySlug: {},
    slugsByLetter: {},
    visibleTermCount: 0,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    let searchTermsData = window.SAILS_LOCALS && _.isArray(window.SAILS_LOCALS.glossaryTerms)
      ? window.SAILS_LOCALS.glossaryTerms
      : [];
    if (_.isArray(searchTermsData)) {
      this.termIndex = searchTermsData.map((term) => {
        return {
          slug: term.slug,
          name: term.name,
          searchableTextLower: [
            term.name,
            term.definition || '',
            term.searchKeywords || '',
          ].join(' ').toLowerCase(),
        };
      });
      this.termBySlug = _.keyBy(this.termIndex, 'slug');
      this.slugsByLetter = _.reduce(this.termIndex, (memo, term) => {
        let letter = term.name.charAt(0).toUpperCase();
        if (!memo[letter]) {
          memo[letter] = [];
        }
        memo[letter].push(term.slug);
        return memo;
      }, {});
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
    // A term card is visible when any indexed text field contains the query
    // (case-insensitive substring).
    termIsVisible: function(slug) {
      let term = this.termBySlug[slug];
      if (!term) {
        // Unknown slug: index and rendered cards have drifted. Hide so the
        // visible-card set stays in sync with visibleTermCount.
        return false;
      }
      let q = (this.searchQuery || '').trim().toLowerCase();
      if (q && term.searchableTextLower.indexOf(q) === -1) {
        return false;
      }
      return true;
    },
    letterIsVisible: function(letter) {
      let slugsForLetter = this.slugsByLetter[letter] || [];
      return _.some(slugsForLetter, (slug) => this.termIsVisible(slug));
    },
    resetFilters: function() {
      this.searchQuery = '';
    },
    // Triggered when the user presses Enter in the search field.
    // Scrolls to the first term whose indexed text matches the current query.
    jumpToFirstHeaderMatch: function() {
      let q = (this.searchQuery || '').trim().toLowerCase();
      if (!q) {
        return;
      }
      let match = _.find(this.termIndex, (t) => t.searchableTextLower.indexOf(q) !== -1);
      if (!match) {
        return;
      }
      let el = document.getElementById('term-' + match.slug);
      if (!el) {
        return;
      }
      // Update the URL hash so the focused term link is shareable.
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
