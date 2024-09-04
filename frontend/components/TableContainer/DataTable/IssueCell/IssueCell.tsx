import React from "react";
import { isEmpty } from "lodash";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import IssuesIndicator from "pages/hosts/components/IssuesIndicator";

interface IIssueCellProps<T> {
  issues: {
    total_issues_count: number;
    critical_vulnerabilities_count?: number;
    failing_policies_count: number;
  };
  rowId: number;
}

const IssueCell = ({ issues, rowId }: IIssueCellProps<any>): JSX.Element => {
  if (isEmpty(issues) || issues.total_issues_count === 0) {
    return <span className="text-muted">{DEFAULT_EMPTY_CELL_VALUE}</span>;
  }

  return (
    <IssuesIndicator
      totalIssuesCount={issues.total_issues_count}
      criticalVulnerabilitiesCount={issues.critical_vulnerabilities_count}
      failingPoliciesCount={issues.failing_policies_count}
      rowId={rowId}
    />
  );
};

export default IssueCell;
