module.exports = {


  friendlyName: 'View device management glossary',


  description: 'Display the device management glossary page, a reference of MDM, osquery, GitOps, and IT security terms optimized for SEO and LLM discoverability.',


  exits: {

    success: {
      viewTemplatePath: 'pages/device-management-glossary'
    },
    badConfig: { responseType: 'badConfig' },

  },


  fn: async function () {

    // Term data is sourced from docs/glossary.yml at build time and exposed at
    // sails.config.builtStaticContent.glossary. See scripts/build-static-content.js.
    if (!_.isObject(sails.config.builtStaticContent) || !_.isArray(sails.config.builtStaticContent.glossary)) {
      throw {badConfig: 'builtStaticContent.glossary'};
    }
    let glossaryTerms = sails.config.builtStaticContent.glossary;

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

    // Slimmer payload for the client: only the fields the search/index uses.
    let glossarySearchTerms = glossaryTerms.map((term) => ({
      slug: term.slug,
      name: term.name,
      definition: term.definition,
      searchKeywords: term.searchKeywords || '',
    }));

    return {
      glossaryTerms,
      glossarySearchTerms,
      termsByLetter,
      alphabet,
      totalTermCount: glossaryTerms.length,
    };

  }


};
