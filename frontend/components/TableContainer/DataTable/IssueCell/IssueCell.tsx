import { isEmpty } from "lodash";
import React from "react";

import IssuesIndicator from "pages/hosts/components/IssuesIndicator";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface IIssueCellProps {
  issues: {
    total_issues_count: number;
    critical_vulnerabilities_count?: number;
    failing_policies_count: number;
  };
  rowId: number;
}

const IssueCell = ({ issues, rowId }: IIssueCellProps): JSX.Element => {
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
