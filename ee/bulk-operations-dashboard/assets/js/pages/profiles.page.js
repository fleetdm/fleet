parasails.registerPage('profiles', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    sortDirection: 'ASC',
    teamFilter: undefined,
    profilesToDisplay: [],
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
    addProfileFormRules: {
      newProfile: {required: true}
    },
    editProfileFormRules: {},
    profileToEdit: {},
    cloudError: '',
    newProfile: undefined,
    syncingMessage: '',
    overlaySyncing: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    this.profilesToDisplay = this.profiles;
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
        this.profilesToDisplay = _.sortByOrder(this.profiles, 'name', 'desc');
      } else {
        this.sortDirection = 'ASC';
        this.profilesToDisplay = _.sortByOrder(this.profiles, 'name', 'asc');
      }
      await this.forceRender();
    },
    changeTeamFilter: async function() {// Used by the team picker dropdown.
      if(this.teamFilter !== undefined){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
          // console.log(profile.profiles);
          return profile.teams && _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
        });
        this.profilesToDisplay = profilesOnThisTeam;
      } else {
        this.profilesToDisplay = this.profiles;
      }
    },
    clickChangeTeamFilter: async function(teamApid) {// Used by the tooltip links.
      this.teamFilter = teamApid;
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
        return profile.teams && _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0;
      });
      this.profilesToDisplay = profilesOnThisTeam;
    },
    clickDownloadProfile: async function(profile) {
      if(!profile.teams){
        window.open('/download-profile?id='+encodeURIComponent(profile.id));
      } else {
        window.open('/download-profile?uuid='+encodeURIComponent(profile.teams[0].uuid));
      }
    },
    clickOpenEditModal: async function(profile) {
      this.profileToEdit = _.clone(profile);
      this.formData.newTeamIds = _.pluck(this.profileToEdit.teams, 'fleetApid');
      this.formData.profile = profile;
      this.modal = 'edit-profile';
    },
    clickOpenDeleteModal: async function(profile) {
      this.formData.profile = _.clone(profile);
      this.modal = 'delete-profile';
    },
    clickOpenAddProfileModal: async function() {
      this.modal = 'add-profile';
    },
    closeModal: async function() {
      this.modal = '';
      this.formErrors = {};
      this.formData = {};
      await this.forceRender();
    },
    submittedForm: async function() {
      this.syncing = false;
      this.closeModal();
    },
    handleSubmittingDeleteProfileForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.deleteProfile.with({profile: argins.profile});
      await this._getProfiles();
    },
    handleSubmittingAddProfileForm: async function() {
      let argins = _.clone(this.formData);
      await Cloud.uploadProfile.with({newProfile: argins.newProfile, teams: argins.teams});
      await this._getProfiles();
    },
    handleSubmittingEditProfileForm: async function() {
      let argins = _.clone(this.formData);
      if(argins.newTeamIds === [undefined]){
        argins.newTeamIds = [];
      }
      await Cloud.editProfile.with({profile: argins.profile, newProfile: argins.newProfile, newTeamIds: argins.newTeamIds});
      await this._getProfiles();
    },
    _getProfiles: async function() {
      this.overlaySyncing = true;
      this.syncingMessage = 'Gathering profiles';
      let newProfilesInformation = await Cloud.getProfiles();
      this.profiles = newProfilesInformation;
      this.overlaySyncing = false;
      await this.changeTeamFilter();
    }
  }
});
