module.exports = {


  friendlyName: 'Walle',


  description: 'Walle something.',


  args: ['cue'],


  inputs: {
    cue: {
      type: 'string',
      required: true
    }
  },


  fn: async function ({cue}) {

    sails.log('Running custom shell script... (`sails run walle`)');

    let report = await sails.helpers.ai.prompt.with({
      expectJson: true,
      prompt: 'Respond only with well-formed JSON (without code fences) in this data shape: `{"scene": "…", "choices": {"…": "…"} }`\nIn `scene`, describe 1-2 paragraphs of this scene played by the character Wall-E from the movie (maintaining a sense of childlike wonder and simple language, keeping it short), based on the following cue: \n```\n'+cue+'\n```\n\nIn the `choices` dictionary, for the key, use a short 2-3 word explanation of a next step of what could happen, and for the value, describe a cue for a next scene to follow up this one.\n\n'
    }).retry();
    let mostRecentPartofScene = report.scene;
    let sceneSoFar = '\n'+mostRecentPartofScene;
    console.log('\n\n'+mostRecentPartofScene+'\n\n'+'What will Wall-E choose?\n',report.choices);

    let upNext = await sails.helpers.ai.decide('What is the best next scene?\n\n```\n'+report.scene+'\n```', report.choices);
    await sails.helpers.process.executeCommand('say <<\'ASDFGHIJK\'\n'+mostRecentPartofScene+'\nASDFGHIJK\n\n');
    console.log('\n* '+upNext+'\n');
    let nextScene = report.choices[upNext];
    await sails.helpers.process.executeCommand('say <<\'ASDFGHIJK\'\n'+nextScene+'\nASDFGHIJK\n\n');
    sceneSoFar += '\n'+nextScene; 

    mostRecentPartofScene = nextScene;
    for (let i=0; i<25; i++) {
      report = await sails.helpers.ai.prompt.with({
        expectJson: true,
        prompt: 'Respond only with well-formed JSON (without code fences) in this data shape: `{"scene": "…", "choices": {"…": "…"} }`\nIn `scene`, continue 1-2 paragraphs of this scene played by the character Wall-E from the movie (maintaining a sense of childlike wonder and simple language, keeping it short), continuing from here: \n```\n'+mostRecentPartofScene+'\n```\n\n…and where the story has taken this turn:\n\n```\n'+nextScene+'\n```\n\nIn the `choices` dictionary, for the key, use a short 2-3 word explanation of a next step of what could happen, and for the value, describe a cue for a next scene to follow up this one.\n\n'
      }).retry();
      console.log('\n\n'+mostRecentPartofScene+'  '+report.scene+'\n\n'+'What will Wall-E choose?\n',report.choices);
      mostRecentPartofScene = report.scene;
      sceneSoFar += '\n'+report.scene;
      let upNext = await sails.helpers.ai.decide('What is the best next scene?\n\n```\n'+report.scene+'\n```', report.choices);
      await sails.helpers.process.executeCommand('say <<\'ASDFGHIJK\'\n'+mostRecentPartofScene+'\nASDFGHIJK\n\n');
      console.log('\n* '+upNext+'\n');
      nextScene = report.choices[upNext];
      await sails.helpers.process.executeCommand('say <<\'ASDFGHIJK\'\n'+nextScene+'\nASDFGHIJK\n\n');
      sceneSoFar += '\n'+nextScene; 
    }//∞
    
    return sceneSoFar;

  }


};

