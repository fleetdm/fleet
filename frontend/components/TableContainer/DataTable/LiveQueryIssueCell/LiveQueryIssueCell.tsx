import React from "react";
import ReactTooltip from "react-tooltip";

import Icon from "components/Icon";

interface ILiveQueryIssueCellProps<T> {
  distributedInterval: number;
  rowId: number;
}

const LiveQueryIssueCell = ({
  distributedInterval,
  rowId,
}: ILiveQueryIssueCellProps<any>): JSX.Element => {
  if (distributedInterval < 60) {
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
          This host might take up to {distributedInterval} to respond.
        </span>
      </ReactTooltip>
    </>
  );
};

export default LiveQueryIssueCell;
