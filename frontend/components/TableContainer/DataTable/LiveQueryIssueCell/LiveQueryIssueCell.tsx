import React from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

interface ILiveQueryIssueCellProps {
  displayName: string;
  distributedInterval: number;
  status: string;
}

const LiveQueryIssueCell = ({
  displayName,
  distributedInterval,
  status,
}: ILiveQueryIssueCellProps): JSX.Element => {
  if (distributedInterval < 60 && status === "online") {
    return <>{displayName}</>;
  }

  return (
    <>
      {displayName}{" "}
      <TooltipWrapper
        tipContent={
          <span className="tooltip__tooltip-text">
            {status === "offline" ? (
              <>
                Offline hosts will not <br />
                respond to a live report.
              </>
            ) : (
              <>
                This host might take up to
                <br /> {distributedInterval} seconds to respond.
              </>
            )}
          </span>
        }
        position="top"
        underline={false}
        showArrow
        tipOffset={8}
      >
        <span className="host-issue tooltip tooltip__tooltip-icon">
          <Icon
            name="error-outline"
            size="small"
            color={status === "offline" ? "status-error" : "status-warning"}
          />
        </span>
      </TooltipWrapper>
    </>
  );
};

export default LiveQueryIssueCell;
