module.exports = {


  friendlyName: 'Get Powershell commands and regenerate queries yaml',


  description: '',



  fn: async function () {
    let path = require('path');
    let YAML = require('yaml');

    let topLvlRepoPath = path.resolve(sails.config.appPath, '../');

    let RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO = 'docs/01-Using-Fleet/standard-query-library/standard-query-library.yml';
    let newYaml = '';

    let yaml = await sails.helpers.fs.read(path.join(topLvlRepoPath, RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO)).intercept('doesNotExist', (err)=>new Error(`Could not find standard query library YAML file at "${RELATIVE_PATH_TO_QUERY_LIBRARY_YML_IN_FLEET_REPO}".  Was it accidentally moved?  Raw error: `+err.message));

    let queries = YAML.parseAllDocuments(yaml).map((yamlDocument)=>{
      let query = yamlDocument.toJSON();
      return query;
    });
    let batchesOfQueries = _.chunk(queries, 5);
    for(let batch of batchesOfQueries) {
      await sails.helpers.flow.simultaneouslyForEach(batch, async (query)=>{
        if(query.kind === 'query'){
          return;
        }
        if(!query.spec.platform.includes('windows')) {
          newYaml += '---\n'+YAML.stringify(query);
          return;
        }
        if(query.powershell){
          return;
        }
        let prompt = `
          Please convert this osquery SQL to a powershell script that writes a comparable result to stdout and does not use osqueryi
          \`\`\`
          ${query.spec.query}
          \`\`\`

          Please return only the powershell script. do not wrap it in any code fences or format it in any way or add any other text.
        `;
        // console.log(prompt);
        let powershellResult = await sails.helpers.ai.prompt.with({prompt:prompt, baseModel: 'o3-mini-2025-01-31'});
        query.spec.powershell = powershellResult;
      });
    }

    // TODO: this regenerates the queries.yml file but does not keep the order.
    await sails.helpers.fs.write(path.join(topLvlRepoPath, 'docs/new.queries.yml'), newYaml, true);

  }


};
