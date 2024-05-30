module.exports = {


  friendlyName: 'Update start CTA visibility',// TODO is there a better name for this?


  description: 'Sets a timestamp to determine when we should show the user the start CTA after they have dismissed it.',


  inputs: {

  },


  exits: {

  },


  fn: async function () {

    let nowAt = Date.now();
    let tomorrowAt = nowAt + (24 * 60 * 60 * 1000);
    this.req.session.expandCtaAt = tomorrowAt;
    return;
  }


};
