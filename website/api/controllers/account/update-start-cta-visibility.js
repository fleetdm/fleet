module.exports = {


  friendlyName: 'Update start CTA visibility',


  description: 'Sets a timestamp to determine when we should show the user the start CTA after they have dismissed it.',


  inputs: {

  },


  exits: {

  },


  fn: async function () {
    if(!this.req.session){
      throw new Error('Consistency violation. When a user sent a request to the update start CTA visibility endpoint, a session is missing.');
    }
    if(this.req.session.expandCtaAt) {
      // Expand the CTA.
      this.req.session.expandCtaAt = 0;
    } else {
      // collapse CTA for 24 hours.
      let nowAt = Date.now();
      let tomorrowAt = nowAt + (24 * 60 * 60 * 1000);
      this.req.session.expandCtaAt = tomorrowAt;
    }
    return;
  }


};
