import Icon from "components/Icon";
import React from "react";
import TooltipWrapper from "components/TooltipWrapper";

interface IPassingColumnHeaderProps {
  isPassing: boolean;
  timeSinceHostCountUpdate: string;
}

const baseClass = "passing-column-header";

const PassingColumnHeader = ({
  isPassing,
  timeSinceHostCountUpdate,
}: IPassingColumnHeaderProps) => {
  const iconName = isPassing ? "success" : "error";
  const columnText = isPassing ? "Yes" : "No";
  const updateText = timeSinceHostCountUpdate
    ? `Host count updated ${timeSinceHostCountUpdate}.`
    : "";

  return (
    <div className={baseClass}>
      <Icon name={iconName} />
      <TooltipWrapper
        tipContent={
          <>
            {updateText}
            <br /> Counts are updated hourly.
          </>
        }
      >
        <span className="status-header-text">{columnText}</span>
      </TooltipWrapper>
    </div>
  );
};

export default PassingColumnHeader;
