parasails.registerPage('admin-hire', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    formData: {
      companyName: '',
      salesforceAccountId: '',
      title: '',
      role: '',
      location: '',
      description: '',
    },
    formErrors: {},
    formRules: {
      companyName: {required: true},
      title: {required: true},
      role: {required: true},
    },
    syncing: false,
    cloudError: '',
    companySuggestions: [],
    showCompanySuggestions: false,
    companySearchTimeout: undefined,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    typeCompanyName: function() {
      // Typing a new name unlinks any previously-picked Salesforce account.
      this.formData.salesforceAccountId = '';
      clearTimeout(this.companySearchTimeout);
      let searchTerm = this.formData.companyName.trim();
      if(searchTerm.length < 2) {
        this.companySuggestions = [];
        return;
      }
      this.companySearchTimeout = setTimeout(async ()=>{
        let matchingAccounts = await Cloud.searchSalesforceAccounts.with({searchTerm});
        // Ignore stale responses if the name changed while we were waiting.
        if(this.formData.companyName.trim() === searchTerm) {
          this.companySuggestions = matchingAccounts;
          this.showCompanySuggestions = true;
        }
      }, 300);
    },

    blurCompanyName: function() {
      this.showCompanySuggestions = false;
    },

    clickSelectCompany: function(account) {
      this.formData.companyName = account.name;
      this.formData.salesforceAccountId = account.salesforceAccountId;
      this.companySuggestions = [];
      this.showCompanySuggestions = false;
    },

    handleSubmittingJobPostingForm: async function(argins) {
      return await Cloud.createJobPosting.with(_.omit(argins, (value)=>{ return value === ''; }));
    },

    submittedJobPostingForm: async function(newJobPosting) {
      this.jobPostings.unshift({...newJobPosting, interestedJobSeekers: []});
      this.formData = {
        companyName: '',
        salesforceAccountId: '',
        title: '',
        role: '',
        location: '',
        description: '',
      };
      this.syncing = false;
      await this.forceRender();
    },

    clickToggleJobPosting: async function(jobPosting) {
      this.syncing = true;
      await Cloud.updateJobPosting.with({jobPostingId: jobPosting.id, isOpen: !jobPosting.isOpen});
      jobPosting.isOpen = !jobPosting.isOpen;
      this.syncing = false;
    },

    changeVerificationStatus: async function(jobSeeker, verificationStatus) {
      this.syncing = true;
      await Cloud.updateJobSeekerVerificationStatus.with({jobSeekerId: jobSeeker.id, verificationStatus});
      jobSeeker.verificationStatus = verificationStatus;
      this.syncing = false;
    },

  }
});
