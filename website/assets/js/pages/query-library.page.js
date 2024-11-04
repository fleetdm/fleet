parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    inputTextValue: '',
    inputTimers: {},
    searchString: '', // The user input string to be searched against the query library
    selectedPlatform: 'macos', // Initially set to 'macos'
  },

  computed: {
    searchResults: function () {
      return this._search(this.policies, this.searchString);
    },

    queriesList: function () {
      return this.searchResults;
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function () {
    //…
  },
  mounted: async function () {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickSelectPlatform(platform) {
      this.selectedPlatform = platform;
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

    _search: function (queries, searchString) {
      if (_.isEmpty(searchString)) {
        return queries;
      }

      const normalize = (value) => _.isString(value) ? value.toLowerCase() : '';
      const searchTerms = normalize(searchString).split(' ');

      return queries.filter((query) => {
        let textToSearch = normalize(query.name) + ', ' + normalize(query.description);
        if (query.contributors) {
          query.contributors.forEach((contributor) => {
            textToSearch += ', ' + normalize(contributor.name) + ', ' + normalize(contributor.handle);
          });
        }
        return (searchTerms.some((term) => textToSearch.includes(term)));
      });
    },
  },

});
