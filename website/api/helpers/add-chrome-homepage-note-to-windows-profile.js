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

    // The Data of each Chrome Startup policy the profile sets, by level and policy name.  A mandatory policy wins
    // over its recommended twin.
    let chromeStartupPolicies = { mandatory: {}, recommended: {} };
    let itemRegExp = /<Item>[\s\S]*?<\/Item>/g;
    let itemMatch;
    while ((itemMatch = itemRegExp.exec(profile)) !== null) {
      let item = itemMatch[0];
      let locUri = ((item.match(/<LocURI>\s*([\s\S]*?)\s*<\/LocURI>/) || [])[1] || '').trim();
      let policyMatch = locUri.match(/\/GoogleChrome~Policy~googlechrome(_recommended)?~Startup(?:_recommended)?\/([^/]+?)(?:_recommended)?$/i);
      if(policyMatch) {
        let level = policyMatch[1] ? 'recommended' : 'mandatory';
        chromeStartupPolicies[level][policyMatch[2].toLowerCase()] = ((item.match(/<Data>([\s\S]*?)<\/Data>/) || [])[1] || '');
      }
    }
    let dataOf = (name)=>{
      return chromeStartupPolicies.mandatory[name] !== undefined ? chromeStartupPolicies.mandatory[name] : chromeStartupPolicies.recommended[name];
    };
    let isEnabled = (name)=>{ return /<enabled\s*\/>/i.test(dataOf(name) || ''); };
    let isDisabled = (name)=>{ return /<disabled\s*\/>/i.test(dataOf(name) || ''); };

    if(!isEnabled('homepagelocation')) {
      return deliveryNotes;
    }
    let newTabPageExplained = /HomepageIsNewTabPage/i.test(deliveryNotes);

    // Startup opens specific pages only in modes 4 and 6, and only the ones in RestoreOnStartupURLs.
    let startupMode = ((dataOf('restoreonstartup') || '').match(/<data\s+id="RestoreOnStartup"\s+value="(\d+)"/i) || [])[1];
    let opensPagesOnStartup = isEnabled('restoreonstartup') && _.contains(['4', '6'], startupMode) && isEnabled('restoreonstartupurls');

    let notes = [];
    if(!isDisabled('homepageisnewtabpage') && !newTabPageExplained) {
      notes.push('Chrome ignores HomepageLocation while the homepage is the New Tab page, which is its default, so disable HomepageIsNewTabPage too.');
    }
    let missing = [];
    if(!isEnabled('showhomebutton')) {
      missing.push('enable ShowHomeButton to show the Home button');
    }
    if(!opensPagesOnStartup) {
      missing.push('set RestoreOnStartup to 4 (Open a list of URLs) and RestoreOnStartupURLs to open the page when Chrome starts');
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
