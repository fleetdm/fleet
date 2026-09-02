parasails.registerPage('configuration-generator', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    generatedOutput: ``,
    parsedItemsInProfile: [],
    deliveryNotes: undefined,
    formData: {
      profileType: 'ddm'
    },
    hideEditButton: false,
    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },
    // Form rules
    formRules: {
      naturalLanguageInstructions: {required: true},
      profileType: {required: true},
    },
    // Syncing / loading state
    syncing: false,
    // Server error state
    cloudError: '',
    filenameOfGeneratedProfile: undefined,
    hasGeneratedProfile: false,
    // Filename and mimetype to fall back on when the generated profile doesn't come with a filename.
    downloadInfoByProfileType: {
      ddm: { filename: 'ddm-command.json', mimeType: 'application/json' },
      mobileconfig: { filename: 'configuration-profile.mobileconfig', mimeType: 'application/x-apple-aspen-config' },
      csp: { filename: 'configuration-profile.xml', mimeType: 'application/xml' },
      // android: { filename: 'android-policy.json', mimeType: 'application/json' },
    },
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝

  beforeMount: function() {
    //…
  },
  mounted: async function() {
    this._setUpAceEditor();
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleSubmittingForm: async function() {
      this.syncing = true;
      io.socket.request({
        method: 'post',
        url: '/api/v1/get-llm-generated-configuration-profile',
        data: {
          profileType: this.formData.profileType,
          naturalLanguageInstructions: this.formData.naturalLanguageInstructions,
        },
        // Socket requests go through the same CSRF check as any other non-GET request, and the
        // sails.io.js client doesn't attach the token on its own the way the Cloud SDK does.
        headers: { 'x-csrf-token': window.SAILS_LOCALS._csrf },
      }, (unusedData, jwr)=>{
        // The generated profile arrives as a broadcast, not as this response, so the only thing
        // worth reading here is a failure -- without it, a rejected request spins forever.
        if(jwr.statusCode >= 300) {
          this._onProfileGenerationError({error: jwr.statusCode});
        }
      });
      // Detach first, so that retrying after an error doesn't leave duplicate listeners attached.
      io.socket.off('profileGenerated', this._onProfileGenerated);
      io.socket.off('error', this._onProfileGenerationError);
      io.socket.on('profileGenerated', this._onProfileGenerated);
      io.socket.on('error', this._onProfileGenerationError);
    },
    _onProfileGenerated: function(response) {
      this.generatedOutput = response.result.profile;
      this.filenameOfGeneratedProfile = response.result.profileFilename;
      this.deliveryNotes = response.result.deliveryNotes;
      this.parsedItemsInProfile = response.result.items;
      this.hasGeneratedProfile = true;
      ace.edit('editor').setValue(response.result.profile);
      this.modal = '';
      this.syncing = false;
      // Disable the socket event listener after we display the results.
      io.socket.off('profileGenerated', this._onProfileGenerated);
    },
    _onProfileGenerationError: function(response) {
      if(!this.syncing) {
        // A failed generation arrives twice: once as a broadcast, and again as the non-2xx
        // response to the request that started it.  Whichever lands first wins.
        return;
      }
      this.cloudError = response.error;
      this.syncing = false;
      io.socket.off('error', this._onProfileGenerationError);
    },
    closeModal: async function() {
      if(!this.syncing){
        this.modal = '';
        await this.forceRender();
      }
    },

    getUpdatedValueFromEditor: function() {
      this.generatedOutput = ace.edit('editor').getValue();
    },
    clickDownloadResult: function() {
      let downloadInfo = this.downloadInfoByProfileType[this.formData.profileType];
      let exportUrl = URL.createObjectURL(new Blob([this.generatedOutput], { type: downloadInfo.mimeType }));
      let exportDownloadLink = document.createElement('a');
      exportDownloadLink.href = exportUrl;
      exportDownloadLink.download = this.filenameOfGeneratedProfile ? this.filenameOfGeneratedProfile : downloadInfo.filename;
      exportDownloadLink.click();
      URL.revokeObjectURL(exportUrl);
    },
    clickEditGeneratedOutput: function() {
      var editor = ace.edit('editor');
      this.hideEditButton = true;
      editor.setReadOnly(false);
    },
    _setUpAceEditor: function() {
      var editor = ace.edit('editor');
      editor.setTheme('ace/theme/fleet');
      editor.session.setMode('ace/mode/xml');
      editor.setOptions({
        minLines: this.minLines ? this.minLines : 20 ,
        maxLines:  this.maxLines ? this.maxLines : 40 ,
      });
      editor.setReadOnly(true);
    },
  }
});
