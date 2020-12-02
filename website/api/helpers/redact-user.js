module.exports = {


  friendlyName: 'Redact user',


  description: 'Destructively remove properties from the provided User record to prepare it for publication.',


  sync: true,


  inputs: {

    user: {
      type: 'ref',
      readOnly: false
    }

  },


  fn: function ({ user }) {
    for (let [attrName, attrDef] of Object.entries(User.attributes)) {
      if (attrDef.protect) {
        delete user[attrName];
      }//ﬁ
    }//∞
  }


};

