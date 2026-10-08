parasails.registerPage('configuration-generator', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    generatedOutput: ``,
    parsedItemsInProfile: [],
    anticipatedItemsInProfile: [],
    anticipatedName: undefined,
    anticipatedDescription: undefined,
    showLoadingOverlay: false,
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
    cloudErrorExplanation: undefined,
    filenameOfGeneratedProfile: undefined,
    hasGeneratedProfile: false,
    // A Windows profile for a third-party app embeds the app's ADMX template, which is hundreds of KB of
    // XML the admin did not ask to read.  The editor shows a placeholder for each one, and the download
    // puts the template back.
    admxTemplatesByPlaceholder: {},
    // Install-only profiles, when the templates are too large to share a file with the policies.
    additionalProfiles: [],
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
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    handleSubmittingForm: async function() {
      console.time('Profile generation');
      this._resetGeneratedProfile();
      this.syncing = true;
      this.hasGeneratedProfile = true;
      this.showLoadingOverlay = true;
      this.$nextTick(()=>{
        this._setUpAceEditor();
      });
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
        console.timeEnd('Profile generation');
        if(jwr.statusCode >= 300) {
          this._onProfileGenerationError({error: jwr.statusCode});
        }
      });
      // Detach first, so that retrying after an error doesn't leave duplicate listeners attached.
      io.socket.off('settingsPreview', this._onSettingsPreview);
      io.socket.off('profileGenerated', this._onProfileGenerated);
      io.socket.off('error', this._onProfileGenerationError);
      io.socket.on('settingsPreview', this._onSettingsPreview);
      io.socket.on('profileGenerated', this._onProfileGenerated);
      io.socket.on('error', this._onProfileGenerationError);
    },
    _resetGeneratedProfile: function() {
      this.generatedOutput = '';
      this.parsedItemsInProfile = [];
      this.anticipatedItemsInProfile = [];
      this.deliveryNotes = undefined;
      this.filenameOfGeneratedProfile = undefined;
      this.cloudErrorExplanation = undefined;
      this.hideEditButton = false;
      this.admxTemplatesByPlaceholder = {};
      this.additionalProfiles = [];
    },
    _onSettingsPreview: function(response) {
      if(!this.syncing) {
        return;
      }
      this.anticipatedItemsInProfile = response.settings;
      this.anticipatedName = response.name;
      this.anticipatedDescription = response.description;
      this.showLoadingOverlay = false;
      io.socket.off('settingsPreview', this._onSettingsPreview);
    },
    _onProfileGenerated: function(response) {
      console.log('Profile generated!: ', response);
      let profileForEditor = this._collapseAdmxTemplates(response.result.profile);
      this.generatedOutput = profileForEditor;
      this.additionalProfiles = response.result.additionalProfiles || [];
      this.showLoadingOverlay = false;
      this.anticipatedItemsInProfile = [];
      this.filenameOfGeneratedProfile = response.result.profileFilename;
      this.deliveryNotes = response.result.deliveryNotes;
      this.parsedItemsInProfile = response.result.items;
      this.hasGeneratedProfile = true;
      let editor = ace.edit('editor');
      editor.setValue(profileForEditor, -1);
      editor.resize(true);
      this.modal = '';
      this.syncing = false;
      io.socket.off('settingsPreview', this._onSettingsPreview);
      io.socket.off('profileGenerated', this._onProfileGenerated);
    },
    _onProfileGenerationError: function(response) {
      if(!this.syncing) {
        // A failed generation arrives twice: once as a broadcast, and again as the non-2xx
        // response to the request that started it.  Whichever lands first wins.
        return;
      }
      if(response.reason) {
        this.cloudErrorExplanation = response.reason;
      }
      this.cloudError = response.error;
      this.syncing = false;
      this.showLoadingOverlay = false;
      this.anticipatedItemsInProfile = [];
      io.socket.off('settingsPreview', this._onSettingsPreview);
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
      let profileToDownload = this.generatedOutput;
      for (let placeholder of Object.keys(this.admxTemplatesByPlaceholder)) {
        profileToDownload = profileToDownload.split(placeholder).join(this.admxTemplatesByPlaceholder[placeholder]);
      }
      this._downloadFile(profileToDownload, this.filenameOfGeneratedProfile ? this.filenameOfGeneratedProfile : downloadInfo.filename, downloadInfo.mimeType);
    },
    clickDownloadAdditionalProfile: function(additionalProfile) {
      this._downloadFile(additionalProfile.profile, additionalProfile.filename, this.downloadInfoByProfileType.csp.mimeType);
    },
    _downloadFile: function(contents, filename, mimeType) {
      let exportUrl = URL.createObjectURL(new Blob([contents], { type: mimeType }));
      let exportDownloadLink = document.createElement('a');
      exportDownloadLink.href = exportUrl;
      exportDownloadLink.download = filename;
      exportDownloadLink.click();
      URL.revokeObjectURL(exportUrl);
    },
    _collapseAdmxTemplates: function(profile) {
      let admxTemplatesByPlaceholder = {};
      let collapsedProfile = String(profile).replace(/<!\[CDATA\[\s*(?:<\?xml[^>]*\?>\s*)?<policyDefinitions[\s\S]*?\]\]>/g, (cdataSection)=>{
        let placeholder = `<![CDATA[ADMX template #${Object.keys(admxTemplatesByPlaceholder).length + 1} (${Math.round(cdataSection.length / 1024)} KB), included when you download this profile]]>`;
        admxTemplatesByPlaceholder[placeholder] = cdataSection;
        return placeholder;
      });
      this.admxTemplatesByPlaceholder = admxTemplatesByPlaceholder;
      return collapsedProfile;
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
      editor.setValue('', -1);
      editor.setReadOnly(true);
      editor.renderer.$fontMetrics.checkForSizeChanges();
      editor.resize(true);
    },
  }
});
