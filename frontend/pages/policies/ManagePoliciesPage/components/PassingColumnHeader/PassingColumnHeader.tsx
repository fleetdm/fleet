import Icon from "components/Icon";
import React from "react";

interface IPassingColumnHeaderProps {
  isPassing: boolean;
}

const baseClass = "passing-column-header";

const PassingColumnHeader = ({ isPassing }: IPassingColumnHeaderProps) => {
  const iconName = isPassing ? "success" : "error";
  const columnText = isPassing ? "Yes" : "No";

  return (
    <div className={baseClass}>
      <Icon name={iconName} />
      <span className="status-header-text">{columnText}</span>
    </div>
  );
};

export default PassingColumnHeader;
