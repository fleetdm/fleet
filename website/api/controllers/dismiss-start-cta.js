module.exports = {


  friendlyName: 'Dismiss start cta',


  description: '',


  inputs: {

  },


  exits: {

  },


  fn: async function ({req, res}) {

    let nowAt = Date.now();
    let tomorrowAt = nowAt + (24 * 60 * 60 * 1000);
    this.req.session.dismissStartCtaUntil = tomorrowAt;
    this.req.session.collapseStartCta = true;

    return;
  }


};
