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
    // Server error state
    cloudError: '',
    showGeneratedQuery: false,
    generatedQueries: {
      macOSQuery: undefined,
      windowsQuery: undefined,
      linuxQuery: undefined,
      chromeOSQuery: undefined,
      macOSCaveats: undefined,
      windowsCaveats: undefined,
      linuxCaveats: undefined,
      chromeOSCaveats: undefined,
    },
    selectedTab: 'macos',
    tablesUsedInQueries: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // this.generatedQueries = {
    //   "macOSQuery": "SELECT name, uuid, version, path, publisher, publisher_id, installed_at, prerelease, uid FROM vscode_extensions;",
    //   "windowsQuery": "SELECT name, uuid, version, path, publisher, publisher_id, installed_at, prerelease, uid FROM vscode_extensions;",
    //   "linuxQuery": "SELECT name, uuid, version, path, publisher, publisher_id, installed_at, prerelease, uid FROM vscode_extensions;",
    //   "chromeOSQuery": "",
    //   "macOSCaveats": "",
    //   "windowsCaveats": "",
    //   "linuxCaveats": "",
    //   "chromeOSCaveats": "The vscode_extensions table is not available on ChromeOS."
    // }
    // this.showGeneratedQuery = true;
  },
  mounted: async function() {
    $('[purpose="copy-button"]').on('click', async function() {
      let code = $(this).closest('[purpose="codeblock"]').find('pre:visible code').text();
      $(this).addClass('copied');
      await setTimeout(()=>{
        $(this).removeClass('copied');
      }, 2000);
      navigator.clipboard.writeText(code);
    });
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
      this.generatedQueries = response.result;
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
      this.generatedQueries = {
        macOSQuery: undefined,
        windowsQuery: undefined,
        linuxQuery: undefined,
        chromeOSQuery: undefined,
        macOSCaveats: undefined,
        windowsCaveats: undefined,
        linuxCaveats: undefined,
        chromeOSCaveats: undefined,
      };
      this.formData.naturalLanguageQuestion = '';
    },
  }
});
