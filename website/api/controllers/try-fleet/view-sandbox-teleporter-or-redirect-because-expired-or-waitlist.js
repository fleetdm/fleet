module.exports = {


  friendlyName: 'View sandbox teleporter or redirect because sandbox expired or waitlist',

  description:
    `Display "Sandbox teleporter" page (an auto-submitting interstitial HTML form used as a hack to grab a bit of HTML
    from the Fleet Sandbox instance, which sets browser localstorage to consider this user logged in and "teleports" them,
    magically authenticated, into their Fleet Sandbox instance running on a different domain), or redirect the user to a
    page about their sandbox instance being expired, or a page explaining that they are on the Fleet Sandbox waitlist.`,

  moreInfoUrl: 'https://github.com/fleetdm/fleet/pull/6380',


  exits: {

    success: {
      viewTemplatePath: 'pages/try-fleet/sandbox-teleporter',
      description: 'This user is being logged into their Fleet Sandbox instance.'
    },

    redirect: {
      description: 'This user does not have a valid Fleet Sandbox instance and is being redirected.',
      responseType: 'redirect'
    },

  },


  fn: async function () {

    if(!this.req.me) {
      throw {redirect: '/try-fleet/login' };
    }

    if(this.req.me.inSandboxWaitlist){
      throw {redirect: '/try-fleet/waitlist' };
    }

    if(!this.req.me.fleetSandboxURL) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxURL has somehow gone missing!`);
    }

    if(!this.req.me.fleetSandboxExpiresAt) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxExpiresAt has somehow gone missing!`);
    }

    if(!this.req.me.fleetSandboxDemoKey) {
      throw new Error(`Consistency violation: The logged-in user's (${this.req.me.emailAddress}) fleetSandboxDemoKey has somehow gone missing!`);
    }

    // If this user's Fleet Sandbox instance is expired, we'll redirect them to the sandbox-expired page
    if(this.req.me.fleetSandboxExpiresAt < Date.now()){
      throw {redirect: '/try-fleet/sandbox-expired' };
    }

    // Respond with view.
    return {
      hideHeaderOnThisPage: true,
    };

  }


};
