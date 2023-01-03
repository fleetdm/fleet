import React from "react";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import Icon from "components/Icon";

import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";

interface IIssueCellProps<T> {
  issues: {
    total_issues_count: number;
    failing_policies_count: number;
  };
  rowId: number;
}

const IssueCell = ({ issues, rowId }: IIssueCellProps<any>): JSX.Element => {
  if (isEmpty(issues) || issues.total_issues_count === 0) {
    return <></>;
  }

  return (
    <>
      <span
        className={`host-issue tooltip tooltip__tooltip-icon`}
        data-tip
        data-for={`host-issue__${rowId.toString()}`}
        data-tip-disable={false}
      >
        <Icon name="issue" />
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor="#3e4771"
        id={`host-issue__${rowId.toString()}`}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          Failing policies ({issues.failing_policies_count})
        </span>
      </ReactTooltip>
      <span className={`total-issues-count`}>{issues.total_issues_count}</span>
    </>
  );
};

export default IssueCell;
