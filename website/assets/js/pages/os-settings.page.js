parasails.registerPage('os-settings', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    selectedPlatform: 'apple', // Initially set to 'macos'
    // Commented-out variables used by configuration profile generator.
    // generatedOutput: ``,
    // ace: undefined,
    // parsedItemsInProfile: [],
    // formData: {
    //   profileType: 'mobileconfig'
    // },
    // // For tracking client-side validation errors in our form.
    // // > Has property set to `true` for each invalid property in `formData`.
    // formErrors: { /* … */ },
    // // Form rules
    // formRules: {
    //   naturalLanguageInstructions: {required: true},
    //   profileType: {required: true},
    // },
    // // Syncing / loading state
    // syncing: false,
    // queryResult: '',
    // // Server error state
    // cloudError: '',
    // modal: '',
    // filenameOfGeneratedProfile: 'Generated profile',
    // hasGeneratedProfile: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    // this._setUpAceEditor();
    if(bowser.windows){
      this.selectedPlatform = 'windows';
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickSelectPlatform: function(platform) {
      this.selectedPlatform = platform;
    },
    // handleSubmittingForm: async function() {
    //   let argins = this.formData;
    //   this.syncing = true;
    //   let generatedResult = await Cloud.getLlmGeneratedConfigurationProfile.with(argins)
    //   .tolerate((err)=>{
    //     this.cloudError = err;
    //     this.syncing = false;
    //   });
    //   if(!this.cloudError) {
    //     this.generatedOutput = generatedResult.profile;
    //     this.filenameOfGeneratedProfile = generatedResult.profileFilename;
    //     this.hasGeneratedProfile = true;
    //     ace.edit('editor').setValue(generatedResult.profile);
    //     this.parsedItemsInProfile = generatedResult.items;
    //     this.modal = '';
    //     this.syncing = false;
    //   }
    // },
    // closeModal: async function() {
    //   if(!this.syncing){
    //     this.modal = '';
    //     await this.forceRender();
    //   }
    // },

    // getUpdatedValueFromEditor: function() {
    //   this.generatedOutput = ace.edit('editor').getValue();
    // },
    // clickDownloadResult: function() {
    //   let exportUrl = URL.createObjectURL(new Blob([this.generatedOutput], { type: 'text/xml;' }));
    //   let exportDownloadLink = document.createElement('a');
    //   exportDownloadLink.href = exportUrl;
    //   // Parse the XML to determine if it is a .mobileconfig or a CSP.
    //   let parser = new DOMParser();
    //   let xmlDoc = parser.parseFromString(this.generatedOutput, 'application/xml');
    //   let hasPlistNode = xmlDoc.getElementsByTagName('plist')[0];
    //   if(!this.filenameOfGeneratedProfile) {
    //     if(hasPlistNode){
    //       exportDownloadLink.download = `Generated configuration profile.mobileconfig`;
    //     } else {
    //       exportDownloadLink.download = 'Generated CSP.xml';
    //     }
    //   } else {
    //     exportDownloadLink.download = this.filenameOfGeneratedProfile;
    //   }
    //   exportDownloadLink.click();
    //   URL.revokeObjectURL(exportUrl);
    // },
    // _setUpAceEditor: function() {
    //   var editor = ace.edit('editor');
    //   editor.setTheme('ace/theme/fleet');
    //   editor.session.setMode('ace/mode/xml');
    //   editor.setOptions({
    //     minLines: this.minLines ? this.minLines : 20 ,
    //     maxLines:  this.maxLines ? this.maxLines : 40 ,
    //   });
    //   editor.setReadOnly(true);
    // },
  }
});
