module.exports = {


  friendlyName: 'Add Chrome homepage note to Windows profile',


  description: 'Add a delivery note to a generated Windows CSP profile that sets Chrome\'s homepage URL without what makes it take effect.',


  extendedDescription:
`Chrome ignores HomepageLocation while the homepage is the New Tab page, which is its default, and the homepage is
only opened by the Home button, which is hidden by default.  A request to "set the Chrome homepage" usually gets
HomepageLocation alone, which is delivered and shows in chrome://policy but changes nothing the user sees.`,


  inputs: {

    profile: {
      type: 'string',
      required: true,
      description: 'The generated Windows CSP profile.'
    },

    deliveryNotes: {
      type: 'string',
      defaultsTo: '',
      description: 'The delivery notes so far.'
    },

  },


  exits: {

    success: {
      outputFriendlyName: 'Delivery notes',
      outputDescription: 'The delivery notes, with the Chrome homepage note added when it applies.',
      outputType: 'string',
    },

  },


  fn: async function ({profile, deliveryNotes}) {

    // The Data of each Chrome Startup policy the profile sets, by policy name, mandatory and recommended alike.
    let chromeStartupPolicies = {};
    let itemRegExp = /<Item>[\s\S]*?<\/Item>/g;
    let itemMatch;
    while ((itemMatch = itemRegExp.exec(profile)) !== null) {
      let item = itemMatch[0];
      let locUri = ((item.match(/<LocURI>\s*([\s\S]*?)\s*<\/LocURI>/) || [])[1] || '').trim();
      let policyMatch = locUri.match(/\/GoogleChrome~Policy~googlechrome(?:_recommended)?~Startup(?:_recommended)?\/([^/]+?)(?:_recommended)?$/i);
      if(policyMatch) {
        chromeStartupPolicies[policyMatch[1].toLowerCase()] = ((item.match(/<Data>([\s\S]*?)<\/Data>/) || [])[1] || '');
      }
    }

    let homepageLocation = chromeStartupPolicies.homepagelocation;
    if(homepageLocation === undefined || /<disabled\s*\/>/i.test(homepageLocation)) {
      return deliveryNotes;
    }
    if(/HomepageIsNewTabPage|New Tab page/i.test(deliveryNotes)) {
      return deliveryNotes;
    }

    let isEnabled = (name)=>{ return chromeStartupPolicies[name] !== undefined && /<enabled\s*\/>/i.test(chromeStartupPolicies[name]); };
    let isDisabled = (name)=>{ return chromeStartupPolicies[name] !== undefined && /<disabled\s*\/>/i.test(chromeStartupPolicies[name]); };

    let notes = [];
    if(!isDisabled('homepageisnewtabpage')) {
      notes.push('Chrome ignores HomepageLocation while the homepage is the New Tab page, which is its default, so disable HomepageIsNewTabPage too.');
    }
    let missing = [];
    if(!isEnabled('showhomebutton')) {
      missing.push('enable ShowHomeButton to show the Home button');
    }
    if(!isEnabled('restoreonstartup')) {
      missing.push('set RestoreOnStartup and RestoreOnStartupURLs to open the page when Chrome starts');
    }
    if(missing.length > 0) {
      notes.push(`The homepage only opens from the Home button, which is hidden by default: ${missing.join(', and ')}.`);
    }
    if(notes.length === 0) {
      return deliveryNotes;
    }
    return _.compact([deliveryNotes].concat(notes)).join(' ');

  }


};
