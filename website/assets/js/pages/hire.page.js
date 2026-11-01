parasails.registerPage('hire', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Server-provided, but absent for logged-out visitors.
    me: undefined,
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
      interestedInJobs: [],
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
    submitted: false,
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    //…
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
    // Fleet's own roles are applied for on their own pages, so only these go in the form.
    jobsAtCustomers: function() {
      return this.jobs.filter((job) => !job.url);
    },
    eligibilityDetailsLabel: function() {
      if(this.formData.eligibilityReason === 'Fleet customer or former customer') {
        return 'Which company uses (or used) Fleet?';
      } else if(this.formData.eligibilityReason === 'Evaluating Fleet Premium (active PoV)') {
        return 'Which company is evaluating Fleet Premium?';
      } else {
        return 'Which workshop? Include the city and date.';
      }
    },
  },

  methods: {

    clickExpressInterest: function(event, job) {
      // Logged-out visitors follow the link to log in.
      if(!this.me) {
        return;
      }
      event.preventDefault();
      let jobLabel = job.jobTitle+' at '+job.company;
      if(!this.formData.interestedInJobs.includes(jobLabel)) {
        this.formData.interestedInJobs.push(jobLabel);
      }
      document.getElementById('looking').scrollIntoView({behavior: 'smooth'});
    },

    typeClearOneFormError: function(field) {
      if(this.formErrors[field]) {
        this.formErrors = _.omit(this.formErrors, field);
      }
    },

    submittedJobSeekerForm: async function() {
      this.submitted = true;
      await this.forceRender();
      document.getElementById('looking').scrollIntoView({behavior: 'smooth'});
    },

  }
});
