module.exports = {


  friendlyName: 'To sentence case',// FUTURE: bring this into machinepack-strings at some point


  description: 'Make a best-effort conversion of the specified text into sentence case.',


  sync: true,


  inputs: {
    text: { type: 'string', example: 'the-Weird Catfood', required: true }
  },


  exits: {
    success: { outputType: 'string', outputExample: 'The weird catfood' },
  },


  fn: function ({ text }) {

    let KNOWN_ACRONYMS = ['JSON', 'REST', 'CLI', 'API', 'FAQ', 'QA', 'UI', 'README'];  // « helps make this smarter about things like: "Fleet rEST aPI" => "Fleet REST API")
    let KNOWN_PROPER_NOUNS = ['Fleet'];// « helps make this smarter about things like: "Deploying fleet" => "Deploying Fleet"

    return text
      .split(/[\s-_]+/)
      .filter((word, idx) => !(idx === 0 && word.match(/^[0-9]+$/))) // « disregard first word if it contains only numbers (this helps capitalization work as expected)
      .map((word, idx)=>{
        if (KNOWN_ACRONYMS.includes(word.toUpperCase())) {
          return word.toUpperCase();
        } else if (idx === 0 || KNOWN_PROPER_NOUNS.includes(word[0].toUpperCase() + word.slice(1).toLowerCase())) {
          return word[0].toUpperCase() + word.slice(1).toLowerCase();
        } else {
          return word.toLowerCase();
        }
      })
      .join(' ');
  }


};

