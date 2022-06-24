parasails.registerPage('register', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // formData: { /* … */ },
    // // For tracking client-side validation errors in our form.
    // // > Has property set to `true` for each invalid property in `formData`.
    // formErrors: { /* … */ },

    // // Form rules
    // formRules: {
    //   emailAddress: {required: true, isEmail: true},
    //   password: {
    //     required: true,
    //     minLength: 12,
    //     // Custom password validation to ensure it contains at least one letter, one number, and one special character. TODO: full list of special characters
    //     custom: (password)=>{
    //       return !! password.match(/[\!\@\#\$\%\^\&\*]/) && password.match(/\d/) && password.match(/\w/);
    //     }
    //   },
    // },
    // // Syncing / loading state
    // syncing: false,
    // // Server error state
    // cloudError: '',
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

    // Using handle-submitting to add sandboxExpirationTimestamp, firstName, and lastName values to our formData before sending it to signup.js
    // handleSubmittingRegisterForm: async function(argins) {
    //   argins.firstName = argins.emailAddress.split('@')[0];
    //   argins.lastName = argins.emailAddress.split('@')[1];
    //   argins.sandboxExpirationTimestamp = new Date(Date.now() + (24*60*60*1000)).toISOString();
    //   await Cloud.signup.with(argins);
    // },

    // After the form is submitted, we'll redirect the user to their Fleet sandbox instance when it is ready
    // submittedRegisterForm: async function() {
    //   this.syncing = true;
    //   let liveFleetSandboxURL = await Cloud.getSandboxStatus.with({id: this.me.id});
    //   window.location = liveFleetSandboxURL;
    // }
  }
});
