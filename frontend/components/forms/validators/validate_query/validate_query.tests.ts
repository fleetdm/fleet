import {
  validateQuery,
  EMPTY_QUERY_ERR,
  expectedSelectErr,
  invalidSyntaxOnLineErr,
} from ".";

const validQueries = [
  "SELECT * FROM users",
  "select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as " +
    "counter from osquery_info i, processes p, time where p.pid = i.pid",
  "select feeds.*, p2.value as sparkle_version from (select a.name as " +
    "app_name, a.path as app_path, a.bundle_identifier as bundle_id, " +
    "p.value as feed_url from (select name, path, bundle_identifier from " +
    "apps) a, preferences p where p.path = a.path || '/Contents/Info.plist' " +
    "and p.key = 'SUFeedURL' and feed_url like 'http://%') feeds left outer " +
    "join preferences p2 on p2.path = app_path || '/Info.plist' where " +
    "(p2.key = 'CFBundleShortVersionString' OR coalesce(p2.key, '') = '')",
];

// osquery only executes SELECT statements, so anything else is invalid even
// when it is well-formed SQLite. Comment-only input belongs here too: it
// contains no statement at all, and would save and then silently do nothing.
const nonSelectQueries = [
  'INSERT INTO users (name) values ("Mike")',
  "CREATE TABLE users (LastName varchar(255))",
  'UPDATE users SET name = "Mike"',
  "DELETE FROM users",
  "DROP TABLE users",
  "ALTER TABLE users ADD COLUMN age int",
  "ATTACH DATABASE '/tmp/other.db' AS other",
  'SELECT 1; INSERT INTO users (name) values ("Mike")',
  "-- just a comment",
  "/* just a comment */",
];

describe("validateQuery", () => {
  it("rejects malformed queries with the error location", () => {
    expect(validateQuery("this is not a thing")).toEqual({
      valid: false,
      error: expectedSelectErr(1),
    });

    const { error, valid } = validateQuery("SELECT * FROM foo bar baz");
    expect(valid).toEqual(false);
    expect(error).toEqual(invalidSyntaxOnLineErr(1, 23));
  });

  it("rejects blank queries", () => {
    const cases = [undefined, "", " ", "   ", "\t", "\n"];
    cases.forEach((query) => {
      const { error, valid } = validateQuery(query);
      expect(valid).toEqual(false);
      expect(error).toEqual(EMPTY_QUERY_ERR);
    });
  });

  it("accepts valid queries", () => {
    validQueries.forEach((query) => {
      const { error, valid } = validateQuery(query);
      expect(valid).toEqual(true);
      expect(error).toBeFalsy();
    });
  });

  it("rejects non-SELECT statements, pointing at the offending line", () => {
    nonSelectQueries.forEach((query) => {
      const { error, valid } = validateQuery(query);
      expect(valid).toEqual(false);
      expect(error).toEqual(expectedSelectErr(1));
    });

    // The location reflects the original text, including leading blank lines
    // and statements after the first semicolon.
    expect(validateQuery("\n\nDROP TABLE users")).toEqual({
      valid: false,
      error: expectedSelectErr(3),
    });
    expect(validateQuery("SELECT 1 FROM users;\nDELETE FROM users")).toEqual({
      valid: false,
      error: expectedSelectErr(2),
    });
  });
});

describe("osquery_sql_parser integration", () => {
  it("#30109 - allow custom escape characters in LIKE clauses", () => {
    const query = `
WITH localusers AS (
      SELECT username, directory || '/.gitconfig' AS gc_path 
      FROM users 
      WHERE 
      shell != '/usr/bin/false'
      AND username NOT LIKE '\\_%' ESCAPE '\\'
      AND username NOT IN ('root', 'person1', 'person2', 'SYSTEM', 'LOCAL SERVICE', 'NETWORK SERVICE')
      AND directory != ''
)
SELECT username, value AS git_signingkey_path
FROM parse_ini
LEFT JOIN localusers ON parse_ini.path=localusers.gc_path
WHERE path IN (SELECT gc_path FROM localusers) AND fullkey = 'user/signingkey';
`;
    const { error, valid } = validateQuery(query);
    expect(valid).toEqual(true);
    expect(error).toBeFalsy();
  });

  it("#34635 - allow VALUES in CTEs, and table names in IN clauses", () => {
    const query = `
-- Step 1: Define config file path suffixes for each supported application
WITH path_suffixes(path) AS (
  VALUES 
    ('/.cursor/mcp.json'), -- Cursor, macOS/Linux/Windows
    ('/Library/Application Support/Claude/claude_desktop_config.json'), -- Claude Desktop, macOS
    ('\\AppData\\Roaming\\Claude\\claude_desktop_config.json'), -- Claude Desktop, Windows
    ('/.claude.json'), -- Claude Code, macOS/Linux
    ('/Library/Application Support/Code/User/mcp.json'), -- VSCode, macOS
    ('/.config/Code/User/mcp.json'), -- VSCode, Linux
    ('\\AppData\\Roaming\\Code\\User\\mcp.json'), -- VSCode, Windows
    ('/.codeium/windsurf/mcp_config.json'), -- Windsurf, macOS
    ('/.gemini/settings.json'), -- Gemini CLI, macOS/Linux/Windows
    ('/.lmstudio/mcp.json') -- LMStudio, macOS/Linux/Windows
), 
-- Step 2: Build full file paths by combining each user's home directory with the path suffixes
full_paths AS (
  SELECT directory || path AS full_path
  FROM users 
  JOIN path_suffixes
), 
-- Step 3: Read config files that exist and concatenate their lines into complete JSON strings
config_files AS (
  SELECT path, group_concat(line, '') AS contents 
  FROM file_lines 
  WHERE path IN full_paths 
  GROUP BY path
) 
-- Step 4: Parse JSON and extract each MCP server configuration
SELECT 
  config_files.path, 
  key AS name, 
  value AS mcp_config 
FROM config_files 
JOIN json_each(
  COALESCE(
    config_files.contents->'$.mcpServers',
    config_files.contents->'$.servers'
    ) -- Most configs use 'mcpServers' key, but some use 'servers' key
)    
`;
    const { error, valid } = validateQuery(query);
    expect(valid).toEqual(true);
    expect(error).toBeFalsy();
  });
});
