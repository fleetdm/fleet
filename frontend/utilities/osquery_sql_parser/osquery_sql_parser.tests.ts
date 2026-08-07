import { astify } from ".";

describe("osquery_sql_parser", () => {
  const shouldParse: Array<[string, string]> = [
    ["basic select", "SELECT * FROM osquery_info;"],
    ["trailing semicolon and whitespace", "SELECT 1 FROM users ;  "],
    [
      "leading whitespace and comments",
      "\n\n-- a comment\n  SELECT 1 FROM users",
    ],
    [
      "join with on clause",
      "SELECT u.username FROM users u JOIN processes p ON u.uid = p.uid;",
    ],
    [
      "cross join with using clause",
      "SELECT u.username, vs.* FROM users u CROSS JOIN vscode_extensions vs USING (uid);",
    ],
    [
      "cte with values and table name in IN clause",
      "WITH sfx(path) AS (VALUES ('/a'),('/b')), fp AS (SELECT directory || path AS f FROM users JOIN sfx) SELECT path FROM file_lines WHERE path IN fp GROUP BY path;",
    ],
    ["json extract operator", "SELECT contents->'$.mcpServers' FROM config;"],
    [
      "custom escape character in LIKE",
      "SELECT * FROM users WHERE username NOT LIKE '\\_%' ESCAPE '\\\\';",
    ],
    ["union", "SELECT name FROM apps UNION SELECT name FROM programs;"],
    ["subselect", "SELECT * FROM (SELECT name FROM apps) a;"],
    [
      "table-valued function",
      "SELECT value FROM plist, json_each(plist.value) WHERE path = '/x';",
    ],
    [
      "window function with empty OVER",
      "SELECT row_number() OVER () FROM users;",
    ],
    ["multiple select statements", "SELECT 1 FROM users; SELECT 2 FROM apps;"],
  ];

  const shouldReject: Array<[string, string]> = [
    ["insert", "INSERT INTO users (name) VALUES ('Mike')"],
    ["create table", "CREATE TABLE users (LastName varchar(255))"],
    ["update", "UPDATE users SET name = 'x'"],
    ["delete", "DELETE FROM users"],
    ["drop", "DROP TABLE users"],
    ["attach", "ATTACH DATABASE '/tmp/x.db' AS x"],
    ["alter", "ALTER TABLE users ADD COLUMN x int"],
    ["show", "SHOW TABLES"],
    ["variable assignment", "@a := 1"],
    ["not sql", "this is not a thing"],
    ["truncated select", "SELECT * FROM foo bar baz"],
    [
      "select followed by insert",
      "SELECT 1; INSERT INTO users (name) VALUES ('x')",
    ],
  ];

  shouldParse.forEach(([label, sql]) => {
    it(`parses ${label}`, () => {
      expect(() => astify(sql)).not.toThrow();
    });
  });

  shouldReject.forEach(([label, sql]) => {
    it(`rejects ${label}`, () => {
      expect(() => astify(sql)).toThrow();
    });
  });

  it("reports syntax error locations in the original text's coordinates", () => {
    try {
      // Leading blank lines must count toward the reported line number so
      // the location matches what the user sees in the editor.
      astify("\n\nSELECT *\nFRM users");
      throw new Error("expected astify to throw");
    } catch (err) {
      const { location } = err as {
        location?: { start: { line: number; column: number } };
      };
      expect(location?.start.line).toEqual(4);
      expect(location?.start.column).toEqual(1);
    }
  });
});
