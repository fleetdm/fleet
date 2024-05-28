parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    inputTextValue: '',
    inputTimers: {},
    searchString: '', // The user input string to be searched against the query library
    selectedKind: 'policy', // Initially set to 'policy', the user may select a different option to filter queries by purpose (e.g., "all queries", "informational", "policies")
    selectedPlatform: 'all platforms', // Initially set to all, the user may select a different option to filter queries by platform (e.g., "all platforms", "macOS", "Windows", "Linux")
    selectedTag: 'built-in', // Initially set to the 'built-in' tag, the user may select a different tag to filter queries by.
  },

  computed: {
    filteredQueries: function () {
      return this.queries.filter(
        (query) =>
          this._isIncluded(query.platform, this.selectedPlatform) &&
          this._isIncluded(query.kind, this.selectedKind) &&
          this._isIncluded(query.tags, this.selectedTag)
      );
    },

    searchResults: function () {
      return this._search(this.filteredQueries, this.searchString);
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
    clickSelectKind(kind) {
      this.selectedKind = kind;
    },

    clickSelectPlatform(platform) {
      this.selectedPlatform = platform;
    },

    clickSelectTag(tag) {
      this.selectedTag = tag;
    },

    clickCard: function (querySlug) {
      this.goto('/queries/' + querySlug); // we can trust the query slug is url-safe
    },

    clickAvatar: function (contributor) {
      this.goto(contributor.htmlUrl);
    },

    getAvatarUrl: function (contributorData) {
      return contributorData ? contributorData.avatarUrl : '';
    },

    getContributorsString: function (contributors) {
      if (!contributors) {
        return;
      }
      const displayName = (contributorData) => {
        if (contributorData) {
          return !contributorData.name
            ? contributorData.handle
            : contributorData.name;
        }
      };
      let contributorString = displayName(contributors[0]);
      if (contributors.length > 2) {
        contributorString += ` and ${contributors.length - 1} others`;
      }
      if (contributors.length === 2) {
        contributorString += ` and ${displayName(contributors[1])}`;
      }
      return contributorString;
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
        if (query.tags) {
          query.tags.forEach((tag) => {
            textToSearch += ', ' + normalize(tag);
          });
        }
        return (searchTerms.some((term) => textToSearch.includes(term)));
      });
    },

    _isIncluded: function (data, selectedOption) {
      if (selectedOption.startsWith('all') || selectedOption === '') {
        return true;
      }
      if (_.isArray(data)) {
        data = data.join(', ');
      }
      return (
        _.isString(data) && data.toLowerCase().includes(selectedOption.toLowerCase())
      );
    },

  },

});
