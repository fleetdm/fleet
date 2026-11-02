parasails.registerPage('superadmins', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // Grant form
    grantFormData: {},
    grantFormErrors: {},
    grantFormRules: {
      emailAddress: {required: true},
    },
    syncingGrant: false,
    grantCloudError: '',
    // Revoke buttons
    syncingRevoke: false,
    revokeCloudError: '',
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
    formatIsoDate: function(timestamp) {
      return moment(timestamp).format('YYYY-MM-DD');
    },
    submittedGrantForm: async function(newSuperAdmin) {
      this.syncingGrant = false;
      this.superAdmins = _.sortBy(this.superAdmins.concat([newSuperAdmin]), 'emailAddress');
      this.revokeCloudError = '';
      this.grantFormData = {};
      await this.forceRender();
    },
    clickRevokeSuperadmin: async function(user) {
      if(!window.confirm(`Remove superadmin access from ${user.emailAddress}?`)) {
        return;
      }
      this.syncingRevoke = true;
      this.revokeCloudError = '';
      try {
        await Cloud.revokeSuperadmin.with({userId: user.id});
        this.superAdmins = this.superAdmins.filter((superAdmin)=>{
          return superAdmin.id !== user.id;
        });
      } catch(err) {
        this.revokeCloudError = (err && err.exit) ? err.exit : 'error';
      }
      this.syncingRevoke = false;
    },
  }
});
