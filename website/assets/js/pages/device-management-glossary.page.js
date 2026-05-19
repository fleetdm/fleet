parasails.registerPage('device-management-glossary-page', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    searchQuery: '',
    activeCategory: 'All',
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
          categories: term.categories,
          searchText: (term.name + ' ' + (term.searchKeywords || '') + ' ' + term.definition).toLowerCase(),
        };
      });
      this.visibleTermCount = this.termIndex.length;
    }
  },

  mounted: function() {
    // Honor a "?q=" or "?category=" param on initial load for shareable filtered views.
    let params = new URLSearchParams(window.location.search);
    let initialQuery = params.get('q');
    let initialCategory = params.get('category');
    if (initialQuery) {
      this.searchQuery = initialQuery;
    }
    if (initialCategory) {
      this.activeCategory = initialCategory;
    }
  },

  //  ╦ ╦╔═╗╔╦╗╔═╗╦ ╦╔═╗╦═╗╔═╗
  //  ║║║╠═╣ ║ ║  ╠═╣║╣ ╠╦╝╚═╗
  //  ╚╩╝╩ ╩ ╩ ╚═╝╩ ╩╚═╝╩╚═╚═╝
  watch: {
    searchQuery: function() {
      this.recomputeVisibleCount();
    },
    activeCategory: function() {
      this.recomputeVisibleCount();
    },
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    termIsVisible: function(slug) {
      let term = _.find(this.termIndex, { slug: slug });
      if (!term) {
        return true;
      }
      // Category filter
      if (this.activeCategory !== 'All' && term.categories.indexOf(this.activeCategory) === -1) {
        return false;
      }
      // Search filter
      let q = (this.searchQuery || '').trim().toLowerCase();
      if (q && term.searchText.indexOf(q) === -1) {
        return false;
      }
      return true;
    },
    letterIsVisible: function(letter) {
      // A letter group is visible if any of its terms are visible.
      let termsForLetter = _.filter(this.termIndex, (t) => t.name.charAt(0).toUpperCase() === letter);
      return _.some(termsForLetter, (t) => this.termIsVisible(t.slug));
    },
    clickCategoryPill: function(category) {
      this.activeCategory = category;
    },
    resetFilters: function() {
      this.searchQuery = '';
      this.activeCategory = 'All';
    },
    recomputeVisibleCount: function() {
      this.visibleTermCount = _.filter(this.termIndex, (t) => this.termIsVisible(t.slug)).length;
    },
  }
});
