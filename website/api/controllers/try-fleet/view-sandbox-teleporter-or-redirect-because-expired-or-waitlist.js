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
    // FUTURE: Remove the route for this controller when all active sandbox instances have expired.

    if(!this.req.me) {
      throw {redirect: '/try-fleet/login' };
    }

    // If the user does not have a Fleet sandbox instance, redirect them to the /fleetctl-preview page.
    if(!this.req.me.fleetSandboxURL || !this.req.me.fleetSandboxExpiresAt || !this.req.me.fleetSandboxDemoKey) {
      throw {redirect: '/try-fleet/fleetctl-preview' };
    }

    // Redirect users with expired sandbox instances to the /fleetctl-preview page.
    if(this.req.me.fleetSandboxExpiresAt < Date.now()){
      throw {redirect: '/try-fleet/fleetctl-preview' };
    }
    // IWMIH, the user has an unexpired Fleet sandbox instance, and will be taken to to the sandbox teleporter page.
    return {
      hideHeaderOnThisPage: true,
    };

  }


};
