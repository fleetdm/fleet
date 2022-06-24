parasails.registerPage('sandbox', {
  //  ╦╔╗╔╦╔╦╗╦╔═╗╦    ╔═╗╔╦╗╔═╗╔╦╗╔═╗
  //  ║║║║║ ║ ║╠═╣║    ╚═╗ ║ ╠═╣ ║ ║╣
  //  ╩╝╚╝╩ ╩ ╩╩ ╩╩═╝  ╚═╝ ╩ ╩ ╩ ╩ ╚═╝
  data: {
    // hasSandbox: false,
    // sandboxInstanceURL: undefined,
    // sandboxIsExpired: false
    // syncing
    // cloud error
  },

  //  ╦  ╦╔═╗╔═╗╔═╗╦ ╦╔═╗╦  ╔═╗
  //  ║  ║╠╣ ║╣ ║  ╚╦╝║  ║  ║╣
  //  ╩═╝╩╚  ╚═╝╚═╝ ╩ ╚═╝╩═╝╚═╝
  beforeMount: function() {
    // If we received a fleetSandboxUrl from the server, set sandboxInstanceURL and hasSandbox
    // (this.sandboxInstanceURL = this.fleetSandboxUrl)
    // (this.hasSandbox = !! this.fleetSandboxUrl)
    // set the sandboxIsExpired flag (this.sandboxIsExpired = this.me.fleetSandboxExpiresAt > Date.now())
  },
  mounted: async function() {
    //…
  },

  //  ╦╔╗╔╔╦╗╔═╗╦═╗╔═╗╔═╗╔╦╗╦╔═╗╔╗╔╔═╗
  //  ║║║║ ║ ║╣ ╠╦╝╠═╣║   ║ ║║ ║║║║╚═╗
  //  ╩╝╚╝ ╩ ╚═╝╩╚═╩ ╩╚═╝ ╩ ╩╚═╝╝╚╝╚═╝
  methods: {

    // For a logged in user who does not have a Fleet sandbox instance
    clickProvisionFleetSandbox: async function() {
      // this.syncing = true;
      // this.sandboxInstanceUrl = await Cloud.provisionFleetSandboxAndRedirect.with({id: this.me.id});
      // window.location = this.sandboxInstanceURL
    },

    // For logged in users who have Fleet sandbox instance that might not be ready yet.
    clickGoToFleetSandbox: async function() {
      // this.syncing = true;
      // this.sandboxInstanceURL = await Cloud.getSandboxStatus.with({id: this.me.id});
      // window.location = this.sandboxInstanceURL
    },

  }
});
