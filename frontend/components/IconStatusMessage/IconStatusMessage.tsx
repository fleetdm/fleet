import React, { ReactNode } from "react";

import classnames from "classnames";
import { IconNames } from "components/icons";
import Icon from "components/Icon/Icon";
import { Colors } from "styles/var/colors";

const baseClass = "icon-status-message";

interface IconStatusMessageProps {
  message: ReactNode;
  iconName?: IconNames;
  iconColor?: Colors;
  className?: string;
  testId?: string;
}

const IconStatusMessage = ({
  message,
  iconName,
  iconColor,
  className,
  testId,
}: IconStatusMessageProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames} data-testid={testId}>
      {iconName && (
        <div className={`${baseClass}__icon`}>
          <Icon name={iconName} color={iconColor} />
        </div>
      )}
      <div className={`${baseClass}__content`}>{message}</div>
    </div>
  );
};

export default IconStatusMessage;
