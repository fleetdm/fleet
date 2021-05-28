parasails.registerPage('query-library', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPurpose: 'all', // Initially set to all, the user may select a different option to filter queries by purpose (e.g., "all queries", "information", "detection")
    selectedPlatform: 'all', // Initially set to all, the user may select a different option to filter queries by platform (e.g., "all platforms", "macOS", "Windows", "Linux")
  },

  computed: {
    filteredQueries: function () {
      return _.filter(this.queries, (query) => this.isIncluded(query.platforms, this.selectedPlatform) && this.isIncluded(query.purpose, this.selectedPurpose));
    },

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
    //…
    clickCard: function (querySlug) {
      window.location = '/sandbox/queries/' + querySlug.toLowerCase(); // TODO remove sandbox from path before deploy to production
    },

    isIncluded: function (queryProperty, selectedOption) {
      if (selectedOption.startsWith('all') || selectedOption === '') {
        return true;
      }
      if (_.isArray(queryProperty)) {
        queryProperty = queryProperty.join(', ');
      }
      return _.isString(queryProperty) && queryProperty.toLowerCase().includes(selectedOption.toLowerCase());
    },

  }

});
