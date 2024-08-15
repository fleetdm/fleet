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
    profiles: [],
    formData: {},
    formErrors: {},
    addProfileFormRules: {
      newProfile: {required: true}
    },
    editProfileFormRules: {
      // no form rules, for this form.
    },
    profileToEdit: {},
    cloudError: '',
    newProfile: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    this.profilesToDisplay = this.profiles;
    console.log(this.teams);
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
    changeTeamFilter: async function() {
      if(this.teamFilter !== undefined){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
          // console.log(profile.profiles);
          return profile.teams && _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
        })
        console.log(profilesOnThisTeam);
        this.profilesToDisplay = profilesOnThisTeam;
      } else {
        this.profilesToDisplay = this.profiles;
      }
    },
    clickChangeTeamFilter: async function(teamApid) {
      this.teamFilter = teamApid;
      console.log(teamApid);
      console.log(this.teamFilter);
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      console.log(this.selectedTeam);
      let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
        return profile.teams && _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
      })
      this.profilesToDisplay = profilesOnThisTeam;
      console.log(profilesOnThisTeam);
    },
    clickDownloadProfile: async function(profile) {
      if(!profile.teams){
        window.open('/download-profile?id='+encodeURIComponent(profile.id));
      } else {
        window.open('/download-profile?uuid='+encodeURIComponent(profile.teams[0].uuid));
      }
      // Call the download profile cloud action
      // Return the downloaded profile with the correct filename
      // Or possible make these just open the download endpoint in a new tab to download it.
    },
    clickOpenEditModal: async function(profile) {
      console.log(profile);
      this.profileToEdit = _.clone(profile);
      this.formData.newTeamIds = _.pluck(this.profileToEdit.teams, 'fleetApid');
      console.log(this.formData.teams);
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
      let response = await Cloud.deleteProfile.with({profile: argins.profile});
      await this._getProfiles();
    },
    handleSubmittingAddProfileForm: async function() {
      let argins = _.clone(this.formData);
      let newProfile = await Cloud.addProfile.with({newProfile: argins.newProfile, teams: argins.teams});
      await this._getProfiles();
    },
    handleSubmittingEditProfileForm: async function() {
      let argins = _.clone(this.formData);
      if(argins.newTeamIds === [undefined]){
        argins.newTeamIds = [];
      }
      let updatedProfile = await Cloud.editProfile.with({profile: argins.profile, newProfile: argins.newProfile, newTeamIds: argins.newTeamIds});
      await this._getProfiles();
    },
    _getProfiles: async function() {
      this.syncing = true;
      let newProfilesInformation = await Cloud.getProfiles();
      this.profiles = newProfilesInformation;
      this.syncing = false;
      await this.changeTeamFilter();
    }
  }
});
