parasails.registerPage('software', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    sortDirection: 'DESC',
    teamFilter: undefined,
    softwareToDisplay: [],
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
    addSoftwareFormRules: {
      newSoftware: {required: true},
    },
    editSoftwareFormRules: {},
    profileToEdit: {},
    cloudError: '',
    newSoftware: undefined,
    showAdvancedOptions: false,
    newSoftwareFilename: undefined,
    syncingMessage: '',
    overlaySyncing: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    this.softwareToDisplay = this.software;
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {
    clickChangeSortDirection: async function() {
      if(this.sortDirection === 'ASC') {
        this.sortDirection = 'DESC';
        this.softwareToDisplay = _.sortByOrder(this.software, 'name', 'desc');
      } else {
        this.sortDirection = 'ASC';
        this.softwareToDisplay = _.sortByOrder(this.software, 'name', 'asc');
      }
      await this.forceRender();
    },
    clickDownloadSoftware: async function(software) {
      if(!software.teams){
        window.open('/download-software?id='+encodeURIComponent(software.id));
      } else {
        window.open('/download-software?fleetApid='+encodeURIComponent(software.teams[0].softwareFleetApid)+'&teamApid='+software.teams[0].fleetApid);
      }
    },
    clickOpenEditModal: async function(software) {
      this.softwareToEdit = software;
      this.modal = 'edit-software';
      let thisSoftware = _.cloneDeep(software);
      this.formData = {
        newTeamIds: _.pluck(thisSoftware.teams, 'fleetApid'),
        software: thisSoftware,
        preInstallQuery: thisSoftware.preInstallQuery,
        installScript: thisSoftware.installScript,
        postInstallScript: thisSoftware.postInstallScript,
        uninstallScript: thisSoftware.uninstallScript,
      };
      // Set isDeployedToAllTeams separately after a 100ms delay This is done to trigger a UI change to disable the teams picker if software is deployed to all future teams.
      await setTimeout(()=>{
        this.$set(this.formData, 'isDeployedToAllTeams', thisSoftware.isDeployedToAllTeams);
      }, 100);
      await this.forceRender();
    },
    clickOpenDeleteModal: async function(software) {
      this.formData.software = _.clone(software);
      this.modal = 'delete-software';
    },
    clickOpenAddSoftwareModal: async function() {
      this.modal = 'add-software';
    },
    changeTeamFilter: async function() {
      if(this.teamFilter !== undefined){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        let softwareOnThisTeam = _.filter(this.software, (software)=>{
          return _.where(software.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
        });
        this.softwareToDisplay = softwareOnThisTeam;
      } else {
        this.softwareToDisplay = this.software;
      }
    },
    submittedForm: async function() {
      this.syncing = false;
      this.closeModal();
    },
    closeModal: async function() {
      this.modal = '';
      this.formErrors = {};
      this.formData = {};
      this.cloudError = '';
      this.showAdvancedOptions = false;
      await this.forceRender();
    },
    clickChangeTeamFilter: async function(teamApid) {
      this.teamFilter = teamApid;
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      let softwareOnThisTeam = _.filter(this.software, (software)=>{
        return _.where(software.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
      });
      this.softwareToDisplay = softwareOnThisTeam;
    },
    handleSubmittingEditSoftwareForm: async function() {
      let argins = _.cloneDeep(this.formData);
      if(argins.newTeamIds[0] === undefined){
        argins.newTeamIds = undefined;
      } else if(argins.isDeployedToAllTeams) {
        argins.newTeamIds = _.pluck(this.teams, 'fleetApid');
      } else {
        argins.newTeamIds = _.uniq(argins.newTeamIds);
      }
      await Cloud.editSoftware.with(argins);
      if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      }
    },
    handleSubmittingAddSoftwareForm: async function() {
      let argins = _.cloneDeep(this.formData);
      if(argins.isDeployedToAllTeams) {
        argins.teams = _.pluck(this.teams, 'fleetApid');
      } else {
        argins.teams = _.uniq(argins.teams);
      }
      await Cloud.uploadSoftware.with(argins);
      if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      }
    },
    handleSubmittingDeleteSoftwareForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.deleteSoftware.with({software: argins.software});
      if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      }
    },
    _getSoftware: async function() {
      this.overlaySyncing = true;
      this.syncingMessage = 'Gathering software';
      let newSoftwareInformation = await Cloud.getSoftware();
      this.software = _.sortByOrder(newSoftwareInformation, 'name', this.sortDirection);
      this.overlaySyncing = false;
      await this.changeTeamFilter();
    }
  }
});
