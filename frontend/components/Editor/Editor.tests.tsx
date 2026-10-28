import { render } from "@testing-library/react";
import { Ace } from "ace-builds";
import React from "react";

import Editor, { EditorMode } from "./Editor";

const renderEditor = (mode: EditorMode): Ace.Editor => {
  let aceEditor: Ace.Editor | undefined;
  render(
    <Editor
      mode={mode}
      value="{}"
      onLoad={(editor) => {
        aceEditor = editor;
      }}
    />
  );
  if (!aceEditor) {
    throw new Error("Editor did not load");
  }
  return aceEditor;
};

describe("Editor", () => {
  const modes: EditorMode[] = ["json", "xml"];

  modes.forEach((mode) => {
    it(`doesn't start Ace's syntax worker in ${mode} mode`, () => {
      const editor = renderEditor(mode);

      expect(editor.getOption("useWorker")).toBe(false);
    });
  });
});
