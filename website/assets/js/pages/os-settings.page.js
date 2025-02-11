parasails.registerPage('os-settings', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    generatedOutput: ``,
    ace: undefined,
    parsedItemsInProfile: [],
    formData: {
      profileType: 'mobileconfig'

    },

    // For tracking client-side validation errors in our form.
    // > Has property set to `true` for each invalid property in `formData`.
    formErrors: { /* … */ },
    modal: '',
    // Form rules
    formRules: {
      naturalLanguageInstructions: {required: true},
      profileType: {required: true},
    },
    // Syncing / loading state
    syncing: false,
    queryResult: '',
    // Server error state
    cloudError: '',
    hasGeneratedProfile: false,
    canDownloadProfile: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝

  watch: {
    generatedOutput: async function (val) {
      if(!this.hasGeneratedProfile){
        let parser = new DOMParser();
        let xmlDoc = parser.parseFromString(val, 'application/xml');
        let results = [];

        // Check if it's a mobileconfig file
        let plistNode = xmlDoc.getElementsByTagName('plist')[0];
        if (plistNode && plistNode.getAttribute('version') === '1.0') {
          // Parse as mobileconfig
          let dictNode = plistNode.getElementsByTagName('dict')[0];
          if (dictNode) {
            // let payloadSettings = {};
            let keys = dictNode.getElementsByTagName('key');
            for (let i = 0; i < keys.length; i++) {
              let keyName = keys[i].textContent.trim();
              let valueNode = keys[i].nextElementSibling;
              if (valueNode) {
                let value = valueNode.textContent.trim();
                results.push({
                  name: keyName,
                  value: value,
                });
              }
            }
          }
        } else {
          // Parse as CSP formatted XML
          let syncMLNode = xmlDoc.getElementsByTagName('SyncML')[0];
          if (syncMLNode) {
            let addNodes = xmlDoc.getElementsByTagName('Add');
            for (let addNode of addNodes) {
              let locURINode = addNode.getElementsByTagName('LocURI')[0];
              let dataNode = addNode.getElementsByTagName('Data')[0];
              if (locURINode && dataNode) {
                let locURI = locURINode.textContent.trim();
                let dataValue = dataNode.textContent.trim();
                results.push({ name: locURI, value: dataValue });
              }
            }
          } else {
            let replaceNodes = xmlDoc.getElementsByTagName('Replace');
            for (let replaceNode of replaceNodes) {
              let itemNodes = replaceNode.getElementsByTagName('Item');
              for (let itemNode of itemNodes) {
                let locURINode = itemNode.getElementsByTagName('LocURI')[0];
                let dataNode = itemNode.getElementsByTagName('Data')[0];
                if (locURINode && dataNode) {
                  let locURI = locURINode.textContent.trim();
                  let dataValue = dataNode.textContent.trim();
                  results.push({ name: locURI, value: dataValue });
                }
              }
            }
          }
        }
        this.canDownloadProfile = results.length > 0;
        this.parsedItemsInProfile = results;
      }
    },


  },
  beforeMount: function() {
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
      let argins = this.formData;
      this.syncing = true;
      let generatedResult = await Cloud.getLlmGeneratedConfigurationProfile.with(argins);
      console.log(generatedResult);
      this.generatedOutput = generatedResult.profile;
      this.hasGeneratedProfile = true;
      ace.edit('editor').setValue(generatedResult.profile);
      this.parsedItemsInProfile = generatedResult.items;
      this.canDownloadProfile = true;
      this.modal = '';
      this.syncing = false;
    },
    closeModal: async function() {
      this.modal = '';
      await this.forceRender();
    },

    getUpdatedValueFromEditor: function() {
      this.generatedOutput = ace.edit('editor').getValue();
    },
    clickDownloadResult: function() {
      let exportUrl = URL.createObjectURL(new Blob([this.generatedOutput], { type: 'text/xml;' }));
      let exportDownloadLink = document.createElement('a');
      exportDownloadLink.href = exportUrl;
      // Parse the XML to determine if it is a .mobileconfig or a CSP.
      let parser = new DOMParser();
      let xmlDoc = parser.parseFromString(this.generatedOutput, 'application/xml');
      let hasPlistNode = xmlDoc.getElementsByTagName('plist')[0];
      if(hasPlistNode){
        exportDownloadLink.download = `Generated configuration profile.mobileconfig`;
      } else {
        exportDownloadLink.download = 'Generated CSP.xml';
      }
      exportDownloadLink.click();
      URL.revokeObjectURL(exportUrl);
    },
    _setUpAceEditor: function() {
      var editor = ace.edit('editor');
      ace.config.setModuleUrl('ace/mode/fleet', '/dependencies/src-min/mode-fleet.js');
      editor.setTheme('ace/theme/fleet');
      editor.session.setMode('ace/mode/xml');
      editor.setOptions({
        minLines: this.minLines ? this.minLines : 20 ,
        maxLines:  this.maxLines ? this.maxLines : 40 ,
      });
    },
  }
});
