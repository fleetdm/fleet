import React from "react";
import ReactTooltip from "react-tooltip";
import { isEmpty } from "lodash";

import Icon from "components/Icon";
import { COLORS } from "styles/var/colors";

const baseClass = "issue-cell";

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
        className={`${baseClass}__icon tooltip tooltip__tooltip-icon`}
        data-tip
        data-for={`host-issue__${rowId.toString()}`}
        data-tip-disable={false}
      >
        <Icon name="error-outline" color="ui-fleet-black-50" size="small" />
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id={`host-issue__${rowId.toString()}`}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          Failing policies ({issues.failing_policies_count})
        </span>
      </ReactTooltip>
      {issues.total_issues_count}
    </>
  );
};

export default IssueCell;
