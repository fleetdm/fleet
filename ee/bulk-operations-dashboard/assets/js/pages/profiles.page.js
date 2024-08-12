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
    addProfileFormRules: {
      // newProfile: {required: true}
    },
    profileToEdit: {},
    cloudError: '',
    newProfile: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
  },
  mounted: async function() {
    this.profilesToDisplay = this.profiles;
    console.log(this.teams)
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
      console.log(this.teamFilter);
      if(this.teamFilter){
        this.selectedTeam = _.find(this.teams, {fleetApid: this.teamFilter});
        console.log(this.selectedTeam);
        let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
          // console.log(profile.profiles);
          return _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
        })
        this.profilesToDisplay = profilesOnThisTeam;
      } else {
        this.profilesToDisplay = this.profiles;
      }
    },
    clickChangeTeamFilter: async function(teamApid) {
      this.teamFilter = teamApid;
      console.log(teamApid);
      this.selectedTeam = _.find(this.teams, {'fleetApid': teamApid});
      console.log(this.selectedTeam);
      let profilesOnThisTeam = _.filter(this.profiles, (profile)=>{
        // console.log(profile.profiles);
        return _.where(profile.teams, {'fleetApid': this.selectedTeam.fleetApid}).length > 0
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
      // this.syncing = false;
      this.profiles = _.remove(this.profiles, (existingProfile)=>{
        return existingProfile.name === argins.profile.name
      })
    },
    handleSubmittingAddProfileForm: async function() {
      let argins = _.clone(this.formData);
      let newProfile = await Cloud.addProfile.with({newProfile: argins.newProfile, teams: argins.teams});
      if(newProfile.teams) {
        for(let team of newProfile.teams){
          console.log(team.fleetApid)
          let thisTeam = _.find(this.teams, {fleetApid: Number(team.fleetApid)});
          team.teamName = thisTeam.teamName;
        }
        console.log(newProfile.teams);
        console.log(newProfile);
        let profileAlreadyExists = _.find(this.profiles, {name: newProfile.name});
        if(profileAlreadyExists){
          this.profiles = _.remove(this.profiles, (existingProfile)=>{
            return existingProfile.name === newProfile.name;
          })
          newProfile.teams = _.merge(profileAlreadyExists.teams, newProfile.teams)
          console.log(this.profiles);
        }
      }
      console.log(this.profiles);
      this.profiles.push(newProfile);
      await this.forceRender;
    },
  }
});
