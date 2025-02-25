module.exports = {


  friendlyName: 'Test llm generated sql',


  description: '',

  inputs: {
    naturalLanguageQuestion: { type: 'string', required: true }
  },


  fn: async function ({naturalLanguageQuestion}) {

    // let assert = require('assert');

    let completeTables = await sails.helpers.getExtendedOsquerySchema();
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
    return lighterTable;
  }))}
    \`\`\`

    Please respond in JSON, with the same data shape as the provided context, but with the array filtered to include only relevant tables.`;
    let filteredTables = await sails.helpers.ai.prompt(schemaFiltrationPrompt, 'gpt-4o', true);



    // Now generate the SQL.
    let sqlPrompt = `Given this question from an IT admin, return osquery SQL I could run on a computer (or fleet of computers) to answer this question.

Here is the question:
\`\`\`
${naturalLanguageQuestion}
\`\`\`

When generating the SQL:
1. Please do not use the SQL "AS" operator, nor alias tables.  Always reference tables by their full name.
2. If this question is related to an application or program, consider using LIKE instead of something verbatim.
3. If this question is not possible to ask given the tables and columns available in the provided context (the osquery schema) for a particular operating system, then use empty string.
4. If this question is a "yes" or "no" question, or a "how many people" question, or a "how many hosts" question, then build the query such that a "yes" returns exactly one row and a "no" returns zero rows.  In other words, if this question is about finding out which hosts match a "yes" or "no" question, then if a host does not match, do not include any rows for it.
5. Use only tables that are supported for each target platform, as documented in the provided context, considering the examples if they exist, and the available columns.
6. For each table that you use, only use columns that are documented for that table, as documented in the provided context.

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
}`;
    let sqlReport = await sails.helpers.ai.prompt(sqlPrompt, 'o1-preview', true);


    // Which of my computers dont have filevault enabled?
    // SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND encrypted = 0 LIMIT 1;

    // Which of my computers do have filevault enabled?
    // SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND encrypted = 1 LIMIT 1;

    // Retrieve a list of all running processes that could establish outbound network connections
    // This seemed to work

    // How many of my devices are on macOS 14?
    // SELECT COUNT(*) FROM os_version WHERE version LIKE '14.%';

    // Retrieve a list of all running processes that have established outbound network connections to remote servers over non-standard ports (not HTTP/HTTPS), including details about the process name, process ID, the user running the process, the remote IP addresses and ports connected to, the MD5 hash of the process executable, and the timestamp of when the connection was established. Exclude any processes that are known system services or signed by trusted vendors. Additionally, only include connections that have been active within the last 24 hours.
    // ```
    // SELECT DISTINCT p.pid, p.name, p.path, p.cmdline, p.start_time AS connection_established, p.md5, np.remote_address, np.remote_port, u.username FROM processes AS p JOIN process_open_sockets AS np ON p.pid = np.pid JOIN users AS u ON p.uid = u.uid WHERE np.remote_port NOT IN (80, 443) AND np.connected = 1 AND p.start_time >= (SELECT strftime('%s','now') - 86400) AND p.signed = 0;
    // ```
    //  no such column: p.md5
    //
    // 2nd try:
    // SELECT processes.name, processes.pid, processes.uid, routes.destination AS remote_ip, routes.gateway AS remote_port, hash.md5, connections.last_connect_time FROM processes JOIN process_open_sockets ON processes.pid = process_open_sockets.pid JOIN routes ON process_open_sockets.remote_address = routes.destination JOIN hash ON processes.path = hash.path JOIN signatures ON processes.path = signatures.path WHERE (routes.gateway NOT IN (80, 443)) AND (CAST((strftime('%s', 'now') - process_open_sockets.remote_address_last_seen) AS INTEGER) < 86400) AND (signatures.trusted = 0 OR signatures.trusted IS NULL);
    // no such table: signatures
    //
    // 3rd try:
    //
    // no such table: signatures
    //
    // 4th try:
    // SELECT DISTINCT processes.name, processes.pid, processes.uid, process_open_sockets.remote_address, process_open_sockets.remote_port, hash.md5, datetime(process_open_sockets.start_time, 'unixepoch') AS connection_time FROM processes JOIN process_open_sockets ON processes.pid = process_open_sockets.pid JOIN hash ON hash.path = processes.path WHERE process_open_sockets.remote_port NOT IN (80, 443) AND NOT EXISTS(SELECT 1 FROM signature WHERE signature.path = processes.path AND signature.is_valid = 1) AND datetime(process_open_sockets.start_time, 'unixepoch') >= datetime('now', '-1 day')
    // no such column: process_open_sockets.start_time


    // console.log('QUESTION:',naturalLanguageQuestion,'\nFILTRATION PROMPT:', schemaFiltrationPrompt, '\nFILTERED TABLES:', filteredTables, '\nSQL PROMPT:', sqlPrompt);


    console.log(sqlReport);



    // Now talk to Fleet.
    // -----
    // TODO: https://github.com/fleetdm/fleet/issues/23037
    // [?] A more complicated example + caveats about this API endpoint: https://github.com/mikermcneil/fleet-at-uber-via-sal-like-dashboard/blob/84660d0a10b4426261bd766400d72bbbaf2e7d5a/scripts/update-reports.js#L159-L218
    // -----

    //   if (!sails.config.custom.fleetBaseUrl || !sails.config.custom.fleetApiToken) {
    //     throw new Error('sails.config.custom.fleetBaseUrl and sails.config.custom.fleetApiToken must both be provided.');
    //   }
    //   let fleetBaseUrl = sails.config.custom.fleetBaseUrl;
    //   let headers = {
    //     Authorization: `Bearer ${sails.config.custom.fleetApiToken}`
    //   };

    //   let httpResponse = await sails.helpers.http.sendHttpRequest.with({
    //   method: 'GET',
    //   url: '/api/v1/fleet/queries/run',
    //   body: {
    //     query_ids: [ specialQueryApid ],//eslint-disable-line camelcase
    //     host_ids: theseApids,//eslint-disable-line camelcase
    //   },
    //   baseUrl: fleetBaseUrl,
    //   headers
    // });



    return sqlReport;


  }


};
