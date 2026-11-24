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
