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

// Renders SQL with table tokens as [name] and column tokens as {name}, using the fleet mode registered by importing SQLEditor.
const markHighlights = (sql: string): string => {
  const tokenizer = new (ace.require("ace/mode/fleet").Mode)().getTokenizer();
  let state: string | string[] = "start";
  return sql
    .split("\n")
    .map((line) => {
      const result = tokenizer.getLineTokens(line, state);
      state = result.state;
      return result.tokens
        .map(({ type, value }: Ace.Token) => {
          if (type === "osquery-token") return `[${value}]`;
          if (type === "osquery-column") return `{${value}}`;
          return value;
        })
        .join("");
    })
    .join("\n");
};

describe("SQLEditor syntax highlighting", () => {
  // os_version, processes, and time are each both a table and a column name.
  it.each([
    "SELECT 1 FROM [os_version] WHERE {major} >= 13",
    "SELECT {time}, {pid} FROM [processes] JOIN [time] WHERE {os_version} > 0",
    "SELECT {name} FROM [processes] proc_alias, [os_version] AS os_alias WHERE {uptime} = 1",
    "SELECT * FROM [users] u LEFT JOIN [processes] ON u.{uid} = {pid}",
    "select * from main.[os_version]",
    "SELECT 1 FROM (\n  SELECT count({time}) FROM [uptime]\n) sub_alias, [os_version] JOIN [processes] WHERE {time} > 0",
    "SELECT 1 FROM /* comment */\n  [os_version] JOIN -- comment\n\n[processes] JOIN   \n  [time]",
  ])("highlights %s", (markedSql) => {
    expect(markHighlights(markedSql.replace(/[[\]{}]/g, ""))).toBe(markedSql);
  });
});
