parasails.registerPage('software', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    sortDirection: 'ASC',
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
      this.softwareToEdit = _.clone(software);
      this.formData.newTeamIds = _.pluck(this.softwareToEdit.teams, 'fleetApid');
      this.formData.software = software;
      this.formData.preInstallQuery = software.preInstallQuery;
      this.formData.installScript = software.installScript;
      this.formData.postInstallScript = software.postInstallScript;
      this.formData.uninstallScript = software.uninstallScript;
      this.modal = 'edit-software';
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


    },
    closeModal: async function() {
      this.modal = '';
      this.formErrors = {};
      this.formData = {};
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
      let argins = _.clone(this.formData);
      if(argins.newTeamIds === [undefined]){
        argins.newTeamIds = [];
      }
      await Cloud.editSoftware.with(argins)
      .tolerate((err)=>{
        this.cloudError = err;
        this.syncing = false;
      });;
      if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      }
    },
    handleSubmittingAddSoftwareForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.uploadSoftware.with({newSoftware: argins.newSoftware, teams: argins.teams})
      // .tolerate((err)=>{
      //   console.log(this.cloudError);
      //   this.cloudError = err;
      //   this.syncing = false;
      // });;
      // if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      // }
    },
    handleSubmittingDeleteSoftwareForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.deleteSoftware.with({software: argins.software})
      .tolerate((err)=>{
        this.cloudError = err;
        this.syncing = false;
      });;
      if(!this.cloudError) {
        this.syncing = false;
        this.closeModal();
        await this._getSoftware();
      }
    },
    _getSoftware: async function() {
      this.syncing = true;
      let newSoftwareInformation = await Cloud.getSoftware();
      this.software = newSoftwareInformation;
      this.syncing = false;
      await this.changeTeamFilter();
    }
  }
});
