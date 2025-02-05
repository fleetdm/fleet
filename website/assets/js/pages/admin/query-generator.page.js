parasails.registerPage('query-generator', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: { /* … */ },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },

    // Form rules
    formRules: {
      naturalLanguageQuestion: {required: true}
    },
    // Syncing / loading state
    syncing: false,
    queryResult: '',
    // Server error state
    cloudError: '',
    showGeneratedQuery: false,
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
    handleSubmittingForm: async function() {
      this.syncing = true;
      io.socket.get('/api/v1/query-generator/get-llm-generated-sql', {naturalLanguageQuestion: this.formData.naturalLanguageQuestion}, ()=>{});
      io.socket.on('queryGenerated', this._onQueryResultsReturned);
      io.socket.on('error', this._onQueryGenerationError);
    },
    _onQueryResultsReturned: function(response) {
      this.queryResult = response.result;
      this.syncing = false;
      this.showGeneratedQuery = true;
      // Disable the socket event listener after we display the results.
      io.socket.off('queryGenerated', this._onQueryResultsReturned);
    },
    _onQueryGenerationError: function(response) {
      this.cloudError = response.error;
      this.syncing = false;
      io.socket.off('error', this._onQueryGenerationError);
    },
    clickResetQueryGenerator: function() {
      this.showGeneratedQuery = false;
      this.formData.naturalLanguageQuestion = '';
    }
  }
});
