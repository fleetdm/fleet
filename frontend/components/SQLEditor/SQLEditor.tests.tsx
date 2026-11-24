import { render } from "@testing-library/react";
import ace, { Ace } from "ace-builds";
import React from "react";

import SQLEditor from "./SQLEditor";

// Ace's popup matches a prefix fuzzily, as a case-insensitive subsequence
// ("dro" matches "build_distro").
const fuzzyMatches = (caption: string, prefix: string): boolean => {
  const lower = caption.toLowerCase();
  let pos = 0;
  return [...prefix.toLowerCase()].every((ch) => {
    pos = lower.indexOf(ch, pos) + 1;
    return pos > 0;
  });
};

// Collects the captions every registered completer offers for a prefix.
const getCompletionCaptions = (prefix: string): string[] => {
  const editor = ace.edit(document.createElement("div"));
  // Exposes language_tools' shared completer list as editor.completers.
  editor.setOptions({ enableBasicAutocompletion: true });
  const captions: string[] = [];
  (editor.completers || []).forEach((completer: Ace.Completer) => {
    completer.getCompletions(
      editor,
      editor.session,
      { row: 0, column: 0 },
      prefix,
      (_err, results) => {
        results.forEach((r) => captions.push(r.caption || r.value || ""));
      }
    );
  });
  return captions.filter((c) => fuzzyMatches(c, prefix));
};

describe("SQLEditor autocomplete", () => {
  it("suggests only the query's table columns when a table is selected", () => {
    render(<SQLEditor value="SELECT * FROM osquery_info" />);

    const captions = getCompletionCaptions("dro");
    expect(captions).toContain("build_distro");
    expect(captions).not.toContain("idrops");
  });

  it("resets to all tables' columns when the editor is cleared", () => {
    const { rerender } = render(
      <SQLEditor value="SELECT * FROM osquery_info" />
    );
    rerender(<SQLEditor value="" />);

    const captions = getCompletionCaptions("dro");
    expect(captions).toContain("build_distro");
    expect(captions).toContain("idrops");
  });
});

// Tokenizes SQL with the fleet mode (registered by importing SQLEditor) and returns each word's token type.
const getTokenTypes = (sql: string): Record<string, string[]> => {
  const { Mode } = ace.require("ace/mode/fleet");
  const tokenizer = new Mode().getTokenizer();
  const types: Record<string, string[]> = {};
  let state: string | string[] = "start";
  sql.split("\n").forEach((line) => {
    const result = tokenizer.getLineTokens(line, state);
    state = result.state;
    result.tokens.forEach((token: Ace.Token) => {
      const word = token.value.trim();
      if (word) {
        types[word] = [...(types[word] || []), token.type];
      }
    });
  });
  return types;
};

describe("SQLEditor syntax highlighting", () => {
  // os_version, processes, and time are each both a table and a column name.
  it.each([
    {
      name: "table after FROM",
      sql: "SELECT 1 FROM os_version WHERE major >= 13",
      expected: { os_version: ["osquery-token"], major: ["osquery-column"] },
    },
    {
      name: "column in SELECT list",
      sql: "SELECT os_version FROM kernel_panics",
      expected: {
        os_version: ["osquery-column"],
        kernel_panics: ["osquery-token"],
      },
    },
    {
      name: "same word as table and column",
      sql: "SELECT time, pid FROM processes JOIN time WHERE processes.pid > 0",
      expected: {
        time: ["osquery-column", "osquery-token"],
        processes: ["osquery-token", "osquery-column"],
      },
    },
    {
      name: "comma-separated tables with aliases",
      sql:
        "SELECT name FROM processes proc_alias, os_version AS os_alias WHERE uptime = 1",
      expected: {
        processes: ["osquery-token"],
        proc_alias: ["identifier"],
        os_version: ["osquery-token"],
        os_alias: ["identifier"],
        uptime: ["osquery-column"],
      },
    },
    {
      name: "LEFT JOIN with ON",
      sql: "SELECT * FROM users u LEFT JOIN processes ON u.uid = processes.uid",
      expected: {
        users: ["osquery-token"],
        processes: ["osquery-token", "osquery-column"],
      },
    },
    {
      name: "table on the line after FROM, with a comment between",
      sql: "select * from /* comment */\n  os_version",
      expected: { os_version: ["osquery-token"] },
    },
    {
      name: "subquery after FROM",
      sql: "SELECT * FROM (SELECT time FROM uptime)",
      expected: { time: ["osquery-column"], uptime: ["osquery-token"] },
    },
  ])("highlights $name", ({ sql, expected }) => {
    const types = getTokenTypes(sql);
    Object.entries(expected).forEach(([word, expectedTypes]) => {
      expect(types[word]).toEqual(expectedTypes);
    });
  });
});
