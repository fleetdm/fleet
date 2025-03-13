module.exports = {


  friendlyName: 'Get llm generated sql',


  description: '',



  inputs: {
    naturalLanguageQuestion: { type: 'string', required: true }
  },


  exits: {
    success: {
      description: 'A SQL query was generated'
    },

    couldNotGenerateQueries: {
      description: 'A set of queries could not be generated for a user using the provided question.',
      responseType: 'badRequest'
    }
  },


  fn: async function ({naturalLanguageQuestion}) {
    // Generate a random room name.
    let roomId = await sails.helpers.strings.random();
    if(this.req.isSocket) {
      // Add the requesting socket to the room.
      sails.sockets.join(this.req, roomId);
    }
    let completeTables = sails.config.builtStaticContent.schemaTables;
    let prunedTables = completeTables.map((table)=>{
      let newTable = _.pick(table,['name','description','platforms', 'examples']);
      newTable.columns = table.columns.map((column) => _.pick(column, ['name', 'description', 'type', 'platforms', 'required']));
      return newTable;
    });

    // Filter down the schema.
    let schemaFiltrationPrompt = `Given this question from an IT admin, and using the provided context (the osquery schema), return the subset of tables that might be relevant for designing an osquery SQL query to answer this question for computers running macOS, Windows, Linux, and/or ChromeOS.

    Here is the question:
    \`\`\`
    ${naturalLanguageQuestion}
    \`\`\`

    Provided context:
    \`\`\`
    ${JSON.stringify(prunedTables.map((table)=>{
    let lighterTable = _.pick(table, ['name','description','platforms']);
    lighterTable.columns = table.columns.map((column)=>{
      let lighterColumn = _.pick(column, ['name', 'description', 'platforms']);
      return lighterColumn;
    });
    return lighterTable;}))}
    \`\`\`

    Please respond in JSON, with the same data shape as the provided context, but with the array filtered to include only relevant tables.


    If no queries can be generated from the provided instructions do not return the datashape above and instead return this JSON in this exact data shape:

    {
      "couldNotGenerateQueries": true
    }`;

    let filteredTables = await sails.helpers.ai.prompt(schemaFiltrationPrompt, 'gpt-4o-mini-2024-07-18', true, 'Please only respond in valid JSON with no codefences or backticks.')
    .intercept((err)=>{
      sails.log.warn(`When trying to get a subset of tables to use to generate a query for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      if(this.req.isSocket){
        // If this request was from a socket and an error occurs, broadcast an 'error' event and unsubscribe the socket from this room.
        sails.sockets.broadcast(roomId, 'error', {error: err});
        sails.sockets.leave(this.req, roomId);
      }
      return 'couldNotGenerateQueries';
    });

    if(filteredTables.couldNotGenerateQueries){
      if(this.req.isSocket){
        sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateQueries'});
        sails.sockets.leave(this.req, roomId);
      } else {
        throw 'couldNotGenerateQueries';
      }
    }

    // 2024-02-26: Testing using a system prompt with a single API request.
    // let systemPrompt = `You are an AI that generates osquery SQL queries for IT admin questions. Use the following osquery schema as context:

    // \`\`\`
    // ${JSON.stringify(prunedTables.map((table)=>{
    // let lighterTable = _.pick(table, ['name','description','platforms']);
    // lighterTable.columns = table.columns.map((column)=>{
    //   let lighterColumn = _.pick(column, ['name', 'description', 'platforms']);
    //   return lighterColumn;
    // });
    // return lighterTable;}))}
    // \`\`\`
    // When generating the SQL:
    // 1. Please do not use the SQL "AS" operator, nor alias tables.  Always reference tables by their full name.
    // 2. If this question is related to an application or program, consider using LIKE instead of something verbatim.
    // 3. If this question is not possible to ask given the tables and columns available in the provided context (the osquery schema) for a particular operating system, then use empty string.
    // 4. If this question is a "yes" or "no" question, or a "how many people" question, or a "how many hosts" question, then build the query such that a "yes" returns exactly one row and a "no" returns zero rows.  In other words, if this question is about finding out which hosts match a "yes" or "no" question, then if a host does not match, do not include any rows for it.
    // 5. Use only tables that are supported for each target platform, as documented in the provided context, considering the examples if they exist, and the available columns.
    // 6. For each table that you use, only use columns that are documented for that table, as documented in the provided context.`;

    // let sqlPrompt = `Given this question from an IT admin, return osquery SQL I could run on a computer (or fleet of computers) to answer this question.

    // Here is the question:
    // \`\`\`
    // ${naturalLanguageQuestion}
    // \`\`\`

    // Please give me all of the above in JSON, with this data shape:

    // {
    //   "macOSQuery": "TODO",
    //   "windowsQuery": "TODO",
    //   "linuxQuery": "TODO",
    //   "chromeOSQuery": "TODO",
    //   "macOSCaveats": "TODO",
    //   "windowsCaveats": "TODO",
    //   "linuxCaveats": "TODO",
    //   "chromeOSCaveats": "TODO",
    // }`;


    // Now generate the SQL.
    let sqlPrompt = `Given this question from an IT admin, return osquery SQL I could run on a computer (or fleet of computers) to answer this question.

    Here is the question:
    \`\`\`
    ${naturalLanguageQuestion}
    \`\`\`

    When generating the SQL:
    1. Please do not use the SQL "AS" operator, nor alias tables.  Always reference tables by their full name.
    2. When generating a query that uses the "LIKE" operator, you should include wildcard characters.
    3. If this question is related to an application or program, consider using LIKE instead of something verbatim.
    4. If this question is not possible to ask given the tables and columns available in the provided context (the osquery schema) for a particular operating system, then use empty string.
    5. If this question is a "yes" or "no" question, or a "how many people" question, or a "how many hosts" question, then build the query such that a "yes" returns exactly one row and a "no" returns zero rows.  In other words, if this question is about finding out which hosts match a "yes" or "no" question, then if a host does not match, do not include any rows for it.
    6. Use only tables that are supported for each target platform, as documented in the provided context, considering the examples if they exist, and the available columns.
    7. For each table that you use, only use columns that are documented for that table, as documented in the provided context.

    Provided context:
    \`\`\`
    ${JSON.stringify(filteredTables)}
    \`\`\`

    Please give me all of the above in JSON, with this data shape:

    {
      "macOSQuery": "TODO",
      "windowsQuery": "TODO",
      "linuxQuery": "TODO",
      "chromeOSQuery": "TODO",
      "macOSCaveats": "TODO",
      "windowsCaveats": "TODO",
      "linuxCaveats": "TODO",
      "chromeOSCaveats": "TODO",
    }


    If no queries can be generated from the provided instructions do not return the datashape above and instead return this JSON in this exact data shape:

    {
      "couldNotGenerateQueries": true
    }`;

    let sqlReport = await sails.helpers.ai.prompt.with({prompt:sqlPrompt, baseModel:'o3-mini-2025-01-31', expectJson: true})
    .intercept((err)=>{
      if(this.req.isSocket){
        // If this request was from a socket and an error occurs, broadcast an 'error' event and unsubscribe the socket from this room.
        sails.sockets.broadcast(roomId, 'error', {error: err});
        sails.sockets.leave(this.req, roomId);
      }
      sails.log.warn(`When trying to generate a query for a user, an error occurred. Full error: ${require('util').inspect(err, {depth: 2})}`);
      return 'couldNotGenerateQueries';
    });

    if(sqlReport.couldNotGenerateQueries){
      if(this.req.isSocket){
        sails.sockets.broadcast(roomId, 'error', {error: 'couldNotGenerateQueries'});
        sails.sockets.leave(this.req, roomId);
      } else {
        throw 'couldNotGenerateQueries';
      }
    }

    // If this request was from a socket, we'll broadcast a 'queryGenerated' event with the sqlReport and unsubscribe the socket
    if(this.req.isSocket){
      sails.sockets.broadcast(roomId, 'queryGenerated', {result: sqlReport});
      sails.sockets.leave(this.req, roomId);
    } else {
      // Otherwise, return the JSON sqlReport.
      return sqlReport;
    }
  }


};
