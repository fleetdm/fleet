parasails.registerPage('hire', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Server-provided, but absent for logged-out visitors.
    me: undefined,
    myJobSeekerProfile: undefined,
    eligibilityReasons: [
      'Fleet customer or former customer',
      'Evaluating Fleet Premium (active PoV)',
      'Fleet-certified (workshop)',
    ],
    formData: {
      rolesSought: [],
      eligibilityReason: '',
      eligibilityDetails: '',
      linkedinUrl: '',
      notes: '',
      interestedInJobPostingIds: [],
    },
    formErrors: {},
    formRules: {
      rolesSought: {custom: (value)=>{ return _.isArray(value) && value.length > 0; }},
      eligibilityReason: {required: true},
      eligibilityDetails: {required: true},
      linkedinUrl: {required: true},
    },
    syncing: false,
    cloudError: '',
    showSavedMessage: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    if(this.myJobSeekerProfile) {
      let openJobPostingIds = _.pluck(this.openJobPostings, 'id');
      this.formData = {
        rolesSought: _.intersection(this.myJobSeekerProfile.rolesSought || [], this.jobBoardRoles),
        eligibilityReason: this.myJobSeekerProfile.eligibilityReason,
        eligibilityDetails: this.myJobSeekerProfile.eligibilityDetails || '',
        linkedinUrl: this.myJobSeekerProfile.linkedinUrl || '',
        notes: this.myJobSeekerProfile.notes || '',
        interestedInJobPostingIds: _.intersection(_.pluck(this.myJobSeekerProfile.interestedInJobPostings, 'id'), openJobPostingIds),
      };
    }
  },
  mounted: async function() {
    if(window.location.hash === '#looking') {
      document.getElementById('looking').scrollIntoView();
    }
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  computed: {
    eligibilityDetailsLabel: function() {
      if(this.formData.eligibilityReason === 'Fleet customer or former customer') {
        return 'Which company uses (or used) Fleet?';
      } else if(this.formData.eligibilityReason === 'Evaluating Fleet Premium (active PoV)') {
        return 'Which company is running the proof of value?';
      } else {
        return 'Which workshop? Include the city and date.';
      }
    },
  },

  methods: {

    clickExpressInterest: function(event, jobPostingId) {
      // Logged-out visitors follow the link to log in.
      if(!this.me) {
        return;
      }
      event.preventDefault();
      if(!this.formData.interestedInJobPostingIds.includes(jobPostingId)) {
        this.formData.interestedInJobPostingIds.push(jobPostingId);
      }
      document.getElementById('looking').scrollIntoView({behavior: 'smooth'});
    },

    typeClearOneFormError: function(field) {
      if(this.formErrors[field]) {
        this.formErrors = _.omit(this.formErrors, field);
      }
    },

    handleSubmittingJobSeekerForm: async function(argins) {
      this.showSavedMessage = false;
      return await Cloud.updateMyJobSeekerProfile.with({...argins, isLooking: true});
    },

    submittedJobSeekerForm: function(savedProfile) {
      this.myJobSeekerProfile = savedProfile;
      this.syncing = false;
      this.showSavedMessage = true;
    },

    clickStopLooking: async function() {
      this.syncing = true;
      this.cloudError = '';
      // Send the saved profile, not unsaved (possibly invalid) edits in the form.
      let savedProfile = await Cloud.updateMyJobSeekerProfile.with({
        ..._.pick(this.myJobSeekerProfile, ['rolesSought', 'eligibilityReason', 'eligibilityDetails', 'linkedinUrl', 'notes']),
        isLooking: false,
      })
      .tolerate((err)=>{
        this.cloudError = err.exit || 'error';
      });
      if(savedProfile) {
        this.myJobSeekerProfile = savedProfile;
        this.formData.interestedInJobPostingIds = [];
      }
      this.syncing = false;
    },

  }
});
