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
    //
  },
  mounted: async function() {
    //
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
    _onQueryResultsReturned: function(response) {
      this.generatedQueries = response.result;
      if(response.result.macOSQuery) {
        this.selectedTab = 'macos';
      } else if(!response.result.macOSQuery){
        this.selectedTab = 'windows';
      } else if(!response.result.windowsQuery){
        this.selectedTab = 'linux';
      } else if(!response.result.linuxQuery) {
        this.selectedTab = 'chromeos';
      } else if(!response.result.chromeOSQuery) {
        this.selectedTab = 'macos';
      }
      this.syncing = false;
      this.showGeneratedQuery = true;
      this._setupCopyButtonEventListener();
      // Disable the socket event listener after we display the results.
      io.socket.off('queryGenerated', this._onQueryResultsReturned);
    },
    _onQueryGenerationError: function(response) {
      this.cloudError = response.error;
      this.syncing = false;
      io.socket.off('error', this._onQueryGenerationError);
    },
    _setupCopyButtonEventListener: function() {
      $('[purpose="copy-button"]').on('click', async function() {
        let code = $(this).closest('[purpose="codeblock"]').find('pre:visible code').text();
        $(this).addClass('copied');
        await setTimeout(()=>{
          $(this).removeClass('copied');
        }, 2000);
        navigator.clipboard.writeText(code);
      });
    }

  }
});
