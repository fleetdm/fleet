import React from "react";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import IssueIcon from "../../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";
interface IIssueCellProps<T> {
  issues: {
    total_issues_count: number;
    failing_policies_count: number;
  };
  rowId: number;
}

const IssueCell = ({
  issues,
  rowId,
}: IIssueCellProps<any>): JSX.Element | null => {
  console.log("issues", issues);

  if (isEmpty(issues)) {
    return null;
  }

  return (
    <>
      <span
        className={`host-issue tooltip__tooltip-icon`}
        data-tip
        data-for={`host-issue__${rowId.toString()}`}
        data-tip-disable={false}
      >
        <img alt="host issue" src={IssueIcon} /> {issues.total_issues_count}
      </span>
      <ReactTooltip
        place="bottom"
        type="dark"
        effect="solid"
        backgroundColor="#3e4771"
        id={`host-issue__${rowId.toString()}`}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          Failing policies ({issues.failing_policies_count})
        </span>
      </ReactTooltip>
    </>
  );
};

export default IssueCell;
