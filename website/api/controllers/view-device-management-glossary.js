module.exports = {


  friendlyName: 'View device management glossary',


  description: 'Display the device management glossary page, a reference of MDM, osquery, GitOps, and IT security terms optimized for SEO and LLM discoverability.',


  exits: {

    success: {
      viewTemplatePath: 'pages/device-management-glossary'
    },

  },


  fn: async function () {

    // Term data lives in its own module so this controller stays small and
    // content edits don't churn this file. See api/datafiles/glossary-terms.js.
    let glossaryTerms = require('../datafiles/glossary-terms');

    // Build a sorted list of letters that actually have terms.
    let lettersWithTerms = _.uniq(glossaryTerms.map((t)=> t.name.charAt(0).toUpperCase())).sort();

    // Build the full A-Z list with active/disabled state for the letter nav.
    let alphabet = [];
    for (let i = 65; i <= 90; i++) {
      let letter = String.fromCharCode(i);
      alphabet.push({
        letter,
        hasTerms: lettersWithTerms.indexOf(letter) !== -1,
      });
    }

    // Group terms by first letter for sectioned rendering.
    let termsByLetter = _.groupBy(glossaryTerms, (t)=> t.name.charAt(0).toUpperCase());

    // Respond with view.
    return {
      glossaryTerms,
      termsByLetter,
      alphabet,
      totalTermCount: glossaryTerms.length,
    };

  }


};
