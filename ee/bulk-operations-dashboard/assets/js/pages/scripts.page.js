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
    formData: {},
    formErrors: {},
    addScriptFormRules: {
      newScript: {required: true},
    },
    editScriptFormRules: {},
    profileToEdit: {},
    cloudError: '',
    newScript: undefined,
    syncingMessage: '',
    overlaySyncing: '',
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
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
      if(this.teamFilter !== undefined){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        let scriptsOnThisTeam = _.filter(this.scripts, (script)=>{
          return _.where(script.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
        });
        this.scriptsToDisplay = scriptsOnThisTeam;
      } else {
        this.scriptsToDisplay = this.scripts;
      }
    },
    clickChangeTeamFilter: async function(teamApid) {
      this.teamFilter = teamApid;
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      let scriptsOnThisTeam = _.filter(this.scripts, (script)=>{
        return _.where(script.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
      });
      this.scriptsToDisplay = scriptsOnThisTeam;
    },
    clickDownloadScript: async function(script) {
      if(!script.teams){
        window.open('/download-script?id='+encodeURIComponent(script.id));
      } else {
        window.open('/download-script?fleetApid='+encodeURIComponent(script.teams[0].scriptFleetApid));
      }
    },
    clickOpenEditModal: async function(script) {
      this.scriptToEdit = _.clone(script);
      this.formData.newTeamIds = _.pluck(this.scriptToEdit.teams, 'fleetApid');
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
      await Cloud.deleteScript.with({script: argins.script});
      await this._getScripts();
    },
    handleSubmittingAddScriptForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.uploadScript.with({newScript: argins.newScript, teams: argins.teams});
      await this._getScripts();
    },
    _getScripts: async function() {
      this.overlaySyncing = true;
      this.syncingMessage = 'Gathering scripts';
      let newScriptsInformation = await Cloud.getScripts();
      this.scripts = newScriptsInformation;
      this.overlaySyncing = false;
      await this.changeTeamFilter();
    }
  }
});
