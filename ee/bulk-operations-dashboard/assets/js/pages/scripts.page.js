parasails.registerPage('scripts', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    sortDirection: 'ASC',
    teamFilter: undefined,
    scriptsToDisplay: [],
    platformFriendlyNames: {
      'darwin': 'macOS, iOS, ipadOS',
      'windows': 'Windows',
      'linux': 'Linux'
    },
    selectedTeam: {},
    modal: '',
    syncing: false,
    profiles: [],
    formData: {},
    formErrors: {},
    addScriptFormRules: {
      newScript: {
        required: true,
      },
      teams: {required: true},
    },
    editScriptFormRules: {
      // no form rules, for this form.
    },
    profileToEdit: {},
    cloudError: '',
    newScript: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    console.log(this.teams)
    this.scriptsToDisplay = this.scripts;
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickChangeSortDirection: async function() {
      if(this.sortDirection === 'ASC') {
        this.sortDirection = 'DESC';
        this.scriptsToDisplay = _.sortByOrder(this.scripts, 'name', 'desc');
      } else {
        this.sortDirection = 'ASC';
        this.scriptsToDisplay = _.sortByOrder(this.scripts, 'name', 'asc');
      }
      await this.forceRender();
    },
    changeTeamFilter: async function() {
      console.log(this.teamFilter);
      if(this.teamFilter !== undefined){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        console.log(this.selectedTeam);
        let scriptsOnThisTeam = _.filter(this.scripts, (script)=>{
          // console.log(script.scripts);
          return _.where(script.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
        })
        this.scriptsToDisplay = scriptsOnThisTeam;
      } else {
        this.scriptsToDisplay = this.scripts;
      }
    },
    clickChangeTeamFilter: async function(teamApid) {
      this.teamFilter = teamApid;
      console.log(teamApid);
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      console.log(this.selectedTeam);
      let scriptsOnThisTeam = _.filter(this.scripts, (script)=>{
        // console.log(script.scripts);
        return _.where(script.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
      })
      this.scriptsToDisplay = scriptsOnThisTeam;
      console.log(scriptsOnThisTeam);
    },
    clickDownloadScript: async function(script) {
      window.open('/download-script?id='+encodeURIComponent(script.teams[0].scriptFleetApid));
      // Call the download script cloud action
      // Return the downloaded script with the correct filename
      // Or possible make these just open the download endpoint in a new tab to download it.
    },
    clickOpenEditModal: async function(script) {
      console.log(script);
      this.scriptToEdit = _.clone(script);
      this.formData.newTeamIds = _.pluck(this.scriptToEdit.teams, 'fleetApid');
      console.log(this.formData.teams);
      this.formData.script = script;
      this.modal = 'edit-script';
    },
    clickOpenDeleteModal: async function(script) {
      this.formData.script = _.clone(script);
      this.modal = 'delete-script';
    },
    clickOpenAddScriptModal: async function() {
      this.modal = 'add-script';
    },
    closeModal: async function() {
      this.modal = '';
      this.formErrors = {};
      this.formData = {};
      await this.forceRender();
    },
    submittedForm: async function() {
      await this._getScripts();
      this.syncing = false;
      this.closeModal();
    },
    handleSubmittingDeleteScriptForm: async function() {
      let argins = _.clone(this.formData);
      let response = await Cloud.deleteScript.with({script: argins.script});
      // this.syncing = false;
      this.scripts = _.remove(this.scripts, (existingScript)=>{
        return existingScript.name === argins.script.name
      })
    },
    handleSubmittingAddScriptForm: async function() {
      let argins = _.clone(this.formData);
      let newScript = await Cloud.addScript.with({newScript: argins.newScript, teams: argins.teams});
    },
    _getScripts: async function() {
      this.syncing = true;
      let newScriptsInformation = await Cloud.getScripts();
      this.scripts = newScriptsInformation;
      this.syncing = false;
      await this.changeTeamFilter();
    }
  }
});
