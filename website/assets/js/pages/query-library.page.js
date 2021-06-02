parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    contributorsDictionary: {},
    inputTextValue: '',
    inputTimers: {},
    searchString: '', // The user input string to be searched against the query library
    selectedPurpose: 'all', // Initially set to all, the user may select a different option to filter queries by purpose (e.g., "all queries", "information", "detection")
    selectedPlatform: 'all', // Initially set to all, the user may select a different option to filter queries by platform (e.g., "all platforms", "macOS", "Windows", "Linux")
  },

  computed: {
    filteredQueries: function () {
      return _.filter(this.queries, (query) => this._isIncluded(query.platforms, this.selectedPlatform) && this._isIncluded(query.purpose, this.selectedPurpose));
    },

    searchResults: function () {
      return this._search(this.filteredQueries, this.searchString);
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
    const uniqueContributors = this._getUniqueContributors(this.queries);
    this.contributorsDictionary = Object.assign({}, await this._threadGitHubAPICalls(uniqueContributors));
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    clickCard: function (querySlug) {
      window.location = '/queries/' + querySlug; // we can trust the query slug is url-safe
    },

    clickAvatar: function(contributor) {
      window.location = contributor.html_url;
    },

    getAvatarUrl: function(contributorData) {
      return contributorData ? contributorData.avatar_url : '';

    },

    getContributorsString: function(contributorHandles, contributorsDictionary) {
      const displayName = (contributorData) => {
        if (contributorData) {
          return !contributorData.name ? contributorData.login : contributorData.name;
        }
      };
      let contributorString = displayName(contributorsDictionary[contributorHandles[0]]);
      if (contributorHandles.length > 2) {
        contributorString += ` and ${contributorHandles.length - 1} others`;
      } 
      if (contributorHandles.length === 2) {
        contributorString += ` and ${displayName(contributorsDictionary[contributorHandles[1]])}`;
      }
      return contributorString;
    },

    delayInput: function(callback, ms, label) {
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

    _search: function (library, searchString) {
      const searchTerms = _.isString(searchString) ? searchString.toLowerCase().split(' ') : [];
      return library.filter((item) => {
        const description = _.isString(item.description)  ? item.description.toLowerCase() : '';
        return _.some(searchTerms, (term) => description.includes(term));
      });
    },

    _isIncluded: function (queryProperty, selectedOption) {
      if (selectedOption.startsWith('all') || selectedOption === '') {
        return true;
      }
      if (_.isArray(queryProperty)) {
        queryProperty = queryProperty.join(', ');
      }
      return _.isString(queryProperty) && queryProperty.toLowerCase().includes(selectedOption.toLowerCase());
    },

    _threadGitHubAPICalls: async function (uniqueContributors) {
      // create threads object with a thread for each contributor
      const threads = uniqueContributors.reduce((threads, contributor) => {
        // each thread is a promise that will resolve when the async call to the GitHub API resolves for that contributor
        threads[contributor] = this._getGitHubUserData(contributor);
        return threads;
      }, {});

      // each thread resolves with a key-value pair where the key is the contributor's GitHub handle and the value is the deserialized JSON response returned by the GitHub API for that contributor
      const resolvedThreads = await Promise.all(Object.keys(threads).map((key) => Promise.resolve(threads[key]).then(result => ({ [key]: result }))))
        .then(resultsArray => {
          const resolvedThreads = resultsArray.reduce((resolvedThreads, result) => {
            Object.assign(resolvedThreads, result);
            return resolvedThreads;
          }, {});
          return resolvedThreads;
        });
      return resolvedThreads;
    },

    _getUniqueContributors: function(queries) {
      return queries.reduce((uniqueContributors, query) => {
        if (query.contributors) {
          uniqueContributors = _.union(uniqueContributors, query.contributors.split(','));
        }
        return uniqueContributors;
      }, []);
    },

    _getGitHubUserData: async function (gitHubHandle) {
      const url = 'https://api.github.com/users/' + encodeURIComponent(gitHubHandle);
      const userData = await fetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      })
      .then(response => response.json())
      .catch(() => {});
      return userData;
    },

  }

});
