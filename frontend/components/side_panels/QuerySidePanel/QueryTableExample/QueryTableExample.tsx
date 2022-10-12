import React from "react";
import { IAceEditor } from "react-ace/lib/types";
import { noop } from "lodash";

import FleetMarkdown from "components/FleetMarkdown";
import FleetAce from "components/FleetAce";

const getExampleDescription = (example: string) => {
  // the text before the first line break is the description of the example.
  const description = example.split("\n")[0];
  return description;
};

const getExampleQuery = (example: string) => {
  // split on the newlines, throw out the first portion, and bring back together with spaces.
  // This lets us get the query example text replacing new line character with spaces.
  const query = example.split("\n").slice(1).join(" ");
  return query;
};

interface IQueryTableExampleProps {
  example: string;
}

const baseClass = "query-table-example";

/**
 * In QueryTableExample we are working with a string in this format:
 * The example description text\nSELECT username, uid\nFROM example_table where uid = 1;
 *
 * The first part is the example description. After that is the first line break, and
 * than the rest of the string is the query. The query also has line breaks that we'd
 * like to replace with space characters.
 */
const QueryTableExample = ({ example }: IQueryTableExampleProps) => {
  const exampleDescription = getExampleDescription(example);
  const exampleQuery = getExampleQuery(example);

  const onEditorBlur = (editor?: IAceEditor) => {
    editor && editor.clearSelection();
  };

  const onEditorLoad = (editor: IAceEditor) => {
    editor.setOptions({
      indentedSoftWrap: false, // removes automatic indentation when wrapping
    });

    // removes focus UI styling
    editor.renderer.visualizeFocus = noop;
  };

  return (
    <div className={baseClass}>
      <h3>Example</h3>
      <FleetMarkdown markdown={exampleDescription} />
      <FleetAce
        wrapperClassName={`${baseClass}__ace`}
        value={exampleQuery}
        showGutter={false}
        onBlur={onEditorBlur}
        onLoad={onEditorLoad}
        style={{ border: "none" }}
        wrapEnabled
        readOnly
      />
    </div>
  );
};

export default QueryTableExample;
