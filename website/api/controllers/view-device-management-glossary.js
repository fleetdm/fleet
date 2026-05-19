// Term data and derived indexes are static. Compute them once at module load
// instead of rebuilding on every request.
let _ = require('@sailshq/lodash');
let glossaryTerms = require('../datafiles/glossary-terms');

let lettersWithTerms = _.uniq(glossaryTerms.map((t)=> t.name.charAt(0).toUpperCase())).sort();

let alphabet = [];
for (let i = 65; i <= 90; i++) {
  let letter = String.fromCharCode(i);
  alphabet.push({
    letter,
    hasTerms: lettersWithTerms.indexOf(letter) !== -1,
  });
}

let termsByLetter = _.groupBy(glossaryTerms, (t)=> t.name.charAt(0).toUpperCase());

// Slimmer payload for the client: only the fields the search/index uses.
let glossarySearchTerms = glossaryTerms.map((term) => ({
  slug: term.slug,
  name: term.name,
  definition: term.definition,
  searchKeywords: term.searchKeywords || '',
}));

let totalTermCount = glossaryTerms.length;


module.exports = {


  friendlyName: 'View device management glossary',


  description: 'Display the device management glossary page, a reference of MDM, osquery, GitOps, and IT security terms optimized for SEO and LLM discoverability.',


  exits: {

    success: {
      viewTemplatePath: 'pages/device-management-glossary'
    },

  },


  fn: async function () {

    return {
      glossaryTerms,
      glossarySearchTerms,
      termsByLetter,
      alphabet,
      totalTermCount,
    };

  }


};
