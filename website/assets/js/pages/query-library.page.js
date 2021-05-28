parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    inputTextValue: '',
    inputTimers: {},
    searchString: '', // The user input string to be searched against the query library
    selectedPurpose: 'all', // Initially set to all, the user may select a different option to filter queries by purpose (e.g., "all queries", "information", "detection")
    selectedPlatform: 'all', // Initially set to all, the user may select a different option to filter queries by platform (e.g., "all platforms", "macOS", "Windows", "Linux")
  },

  computed: {
    filteredQueries: function () {
      return _.filter(this.queries, (query) => this.isIncluded(query.platforms, this.selectedPlatform) && this.isIncluded(query.purpose, this.selectedPurpose));
    },

    searchResults: function () {
      return this.search(this.filteredQueries, this.searchString);
    },

    queriesList: function () {
      return this.searchResults;
    }

  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    isIncluded: function (queryProperty, selectedOption) {
      if (selectedOption.startsWith('all') || selectedOption === '') {
        return true;
      }
      if (_.isArray(queryProperty)) {
        queryProperty = queryProperty.join(', ');
      }
      return _.isString(queryProperty) && queryProperty.toLowerCase().includes(selectedOption.toLowerCase());
    },

    search: function (library, searchString) {
      const searchTerms = _.isString(searchString) ? searchString.toLowerCase().split(' ') : [];
      return library.filter((item) => {
        const description = _.isString(item.description)  ? item.description.toLowerCase() : '';
        return _.some(searchTerms, (term) => description.includes(term));
      });
    },

    setSearchString: function () {
      this.searchString = this.inputTextValue;
    },

    delayInput: function(callback, ms, label) {
      let inputTimers = this.inputTimers;
      return function () {
        label = label || 'defaultTimer';
        _.has(inputTimers, label) ? clearTimeout(inputTimers[label]) : 0;
        inputTimers[label] = setTimeout(callback, ms);
      };
    },

    clickCard: function (querySlug) {
      window.location = '/sandbox/queries/' + querySlug.toLowerCase(); // TODO remove sandbox from path before deploy to production
    },

  }

});
