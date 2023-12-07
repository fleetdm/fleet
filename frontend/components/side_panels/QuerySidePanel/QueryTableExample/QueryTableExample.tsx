import React from "react";

import FleetMarkdown from "components/FleetMarkdown";

interface IQueryTableExampleProps {
  example: string;
}

const baseClass = "query-table-example";

const QueryTableExample = ({ example }: IQueryTableExampleProps) => {
  return (
    <div className={baseClass}>
      <h3>Example</h3>
      <FleetMarkdown markdown={example} name="query-table-example" />
    </div>
  );
};

export default QueryTableExample;
