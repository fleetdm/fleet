module.exports = {


  friendlyName: 'Walle',


  description: 'Walle something.',


  args: ['cue'],


  inputs: {
    
    cue: {
      type: 'string',
      required: true
    },

    n: {
      type: 'number',
      min: 2,
      max: 25,
      defaultsTo: 3,
    },

  },


  fn: async function ({cue, n}) {

    sails.log('Running custom shell script... (`sails run walle`)');

    let wpm = 100;
    let synopsis = await sails.helpers.ai.prompt('Come up with a 1-2 sentence synopsis for a Wall-E themed adventure that will have an intriguing beginning, a climax, and finally a conclusion, which will be over '+n+'-'+n*2+' paragraphs long.  (Remember, the synopsis you are generating is no longer than 1 short paragraph.)').retry();
    sails.log.info('[Raw synopsis] '+synopsis+'\n');
    synopsis = await sails.helpers.ai.satisfy(synopsis, ['just the string literal, not an object', 'no longer than 1 short paragraph']);
    sails.log('[Confirmed synopsis] '+((typeof synopsis === 'string')?synopsis : require('util').inspect(synopsis))+'\n');
    let getRememberThePlotStatement = (i) => '\n\nRemember, this is a story involving the character Wall-E from the movie (maintaining a sense of childlike wonder and simple language).  Here is the overall synopsis, where we\'re approximately at paragraph '+i+'/'+n+':\n\n```\n'+synopsis+'\n```\n\nMake sure this next development in the story stays on track, and roughly matches our position in the story (after this part, the story will be '+Math.floor(i/n*100)+'% written).  Over the course of the story, it should have an intriguing beginning, a climax, and finally a conclusion.  For example, if this is the final part of the story (i.e. 100%), then you MUST write the ending so that it matches the synopsis.\n';

    let report = await sails.helpers.ai.prompt.with({
      expectJson: true,
      prompt: 'Respond only with well-formed JSON (without code fences) in this data shape: `{"scene": "…", "choices": {"…": "…"} }`\nIn `scene`, describe 1-2 paragraphs of this scene based on the following cue: \n```\n'+cue+'\n```\n\n'+getRememberThePlotStatement(1)+'\n\nIn the `choices` dictionary, for the keys, use short 2-3 word explanations of a next step of what could happen, and for each value, describe the corresponding cue for a next scene to follow up this one.\n\n'
    }).retry();
    let mostRecentPartofScene = report.scene;
    let sceneSoFar = mostRecentPartofScene;
    sails.log(mostRecentPartofScene+'\n\n'+'What will Wall-E choose?\n - '+Object.keys(report.choices).join('\n - '));

    let upNext;
    await sails.helpers.flow.simultaneously([
      async ()=>{
        upNext = await sails.helpers.ai.decide('What is the most appropriate and/or interesting next scene from here?\n\n```\n'+report.scene+'\n```\n\n'+getRememberThePlotStatement(1), report.choices);
        sails.log('* '+upNext+'\n');
      },
      async ()=>{
        await sails.helpers.process.executeCommand('say --rate='+wpm+' <<\'ASDFGHIJK\'\n'+mostRecentPartofScene+'\nASDFGHIJK\n\n');
      }
    ]);
    let nextSceneCue = report.choices[upNext];


    for (let i=0; i<n; i++) {
      report = await sails.helpers.ai.prompt.with({
        expectJson: true,
        prompt: 'Respond only with well-formed JSON (without code fences) in this data shape: `{"scene": "…", "choices": {"…": "…"} }`\nIn `scene`, write the next 1-2 paragraphs of the story, advancing the plot and embellishing.  The part of the scene immediately previous went like this: \n```\n'+mostRecentPartofScene+'\n```\n\nHere is a summary cue for what is supposed to happen next and what you are supposed to write 1-2 paragraphs about:\n\n```\n'+nextSceneCue+'\n```\n\n'+getRememberThePlotStatement(i+1)+'\n\nAlso remember: Do NOT use the summary cue in your response; you need to write it yourself.\n\nIn the `choices` dictionary, include at least 2 and no more than 5 keys.  For the keys, use a short 2-3 word explanation of a next step of what could happen, and for each value, describe the corresonding cue for a next scene to follow up this one.\n\n'
      }).retry();
      sails.log('['+(i+1)+'/'+(n)+'] '+report.scene+'\n\n'+'What will Wall-E choose?\n - '+Object.keys(report.choices).join('\n - '));
      mostRecentPartofScene = report.scene;
      sceneSoFar += '\n\n'+report.scene;
      let upNext;
      await sails.helpers.flow.simultaneously([
        async()=>{
          upNext = await sails.helpers.ai.decide('What is the most appropriate and/or interesting next scene from here?\n\n```\n'+report.scene+'\n```\n\n'+getRememberThePlotStatement(i+1), report.choices);
          sails.log('* '+upNext+'\n');
        },
        async()=>{
          await sails.helpers.process.executeCommand('say --rate='+wpm+' <<\'ASDFGHIJK\'\n'+mostRecentPartofScene+'\nASDFGHIJK\n\n');
          if (i+1 < n) {
            await sails.helpers.process.executeCommand('say -v daniel --rate='+wpm+' <<\'ASDFGHIJK\'\nWhat will happen next?\nASDFGHIJK\n\n');
            await sails.helpers.flow.pause(2000);
          }
        }
      ]);
      nextSceneCue = report.choices[upNext];
      await sails.helpers.process.executeCommand('say -v agnes --rate='+wpm+' <<\'ASDFGHIJK\'\n'+upNext+'\nASDFGHIJK\n\n');

    }//∞

    let finalScene = await sails.helpers.ai.prompt.with({
      prompt: 'Write the final 1-2 paragraphs of the story (synopsis is included at the end of this prompt).  The part of the scene immediately previous to this finale you are writing went like this: \n```\n'+mostRecentPartofScene+'\n```\n\nHere is a summary cue for what WAS supposed to happen next and what you are supposed to write 1-2 paragraphs about to finish the story (but make sure it actually makes sense with the synopsis and finishing the plot.  Prioritize providing the appropriate conclusion as stated in the synopsis):\n\n```\n'+nextSceneCue+'\n```\n\n'+getRememberThePlotStatement(n)+'\n\n'
    }).retry();
    sails.log('\n\n[Ending] '+finalScene+'\n\n');
    sceneSoFar += '\n\n'+finalScene;
    
    return sceneSoFar;

  }


};

