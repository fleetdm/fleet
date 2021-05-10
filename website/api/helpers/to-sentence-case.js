module.exports = {


  friendlyName: 'To sentence case',


  description: 'Make a best-effort conversion of the specified text into sentence case.',


  sync: true,


  inputs: {
    text: { type: 'string', example: 'the-Weird Catfood', required: true }
  },


  exits: {
    success: { outputType: 'string', outputExample: 'The weird catfood' },
  },


  fn: function ({ text }) {
    return text
      .split(/[\s-_]+/)
      .filter((word, idx) => !(idx === 0 && word.match(/[0-9]+/))) // « strip off any leading numbers so first word is actually capitalized
      .map((word, idx) => (idx === 0? word[0].toUpperCase() : word[0].toLowerCase())+word.slice(1))
      .join(' ');
  }


};

