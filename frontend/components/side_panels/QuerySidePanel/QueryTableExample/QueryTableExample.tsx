import React from "react";
import { IAceEditor } from "react-ace/lib/types";
import { noop } from "lodash";

import FleetMarkdown from "components/FleetMarkdown";
import FleetAce from "components/FleetAce";

const getExampleDescription = (example: string) => {
  // the text before the from the first split is the description of the example.
  const description = example.split("\n```\n")[0];
  return description;
};

const getExampleQueries = (example: string) => {
  // split on the newlines, throw out the first portion, and bring back together with spaces.
  // This lets us get the query example text replacing new line character with spaces.
  const queries = example.split("```").slice(1, -1);
  return queries.map((query) => {
    return query.replace(/\n/g, "");
  });
};

// seperate examples are separated by \n```\n\n string. We can split on that and then
// pull out the description and queries from this.
const generateExamples = (exampleStr: string) => {
  const examples = exampleStr.split("\n```\n\n");
  return examples.map((example) => ({
    description: getExampleDescription(example),
    queries: getExampleQueries(example),
  }));
};

interface IQueryTableExampleProps {
  example: string;
}

const baseClass = "query-table-example";

/**
 * In QueryTableExample we are working with a string in this format:
 * The example description text\n```\nSELECT username, uid FROM example_table where uid = 1 \n```;
 *
 * The first part is the example description. After that is the first line break, and
 * than the rest of the string is the query. The query also has line breaks that we'd
 * like to replace with space characters.
 */
const QueryTableExample = ({ example }: IQueryTableExampleProps) => {
  const examples = generateExamples(example);

  console.log(examples);
  // const exampleDescription = getExampleDescription(example);
  // const exampleQuery = getExampleQuery(example);

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
      {examples.map((exampleSet) => (
        <div className={`${baseClass}__example-set`}>
          <FleetMarkdown markdown={exampleSet.description} />
          {exampleSet.queries.map((query) => (
            <FleetAce
              wrapperClassName={`${baseClass}__ace-display`}
              value={query}
              showGutter={false}
              onBlur={onEditorBlur}
              onLoad={onEditorLoad}
              style={{ border: "none" }}
              wrapEnabled
              readOnly
            />
          ))}
        </div>
      ))}
    </div>
  );
};

export default QueryTableExample;
