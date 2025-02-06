import React from "react";
import { kebabCase } from "lodash";
import { internationalNumberFormat } from "utilities/helpers";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";
import Card from "components/Card";

interface IHostCountCard {
  count: number;
  title: string;
  iconName: IconNames;
  path: string;
  tooltip?: string;
  notSupported?: boolean;
  className?: string;
  iconPosition?: "top" | "left";
}

const baseClass = "host-count-card";

const HostCountCard = ({
  count,
  title,
  iconName,
  path,
  tooltip,
  notSupported = false,
  className,
  iconPosition = "top",
}: IHostCountCard) => {
  // Renders opaque information as host information is loading

  const classes = classnames(`${baseClass}__card`, `${kebabCase(title)}-card`, {
    [`${baseClass}__not-supported`]: notSupported,
    [`${className}`]: !!className,
  });

  const renderIcon = () => (
    <Icon
      name={iconName}
      size="large-card"
      color="ui-fleet-black-75"
      className={`${baseClass}__card-icon`}
    />
  );

  const renderCount = () => {
    return notSupported ? (
      <div className={`${baseClass}__not-supported-text`}>Not supported</div>
    ) : (
      <div
        className={`${baseClass}__count ${baseClass}__count--${kebabCase(
          title
        )}`}
      >
        {internationalNumberFormat(count)}
      </div>
    );
  };

  const renderDescription = () => {
    return (
      <div className={`${baseClass}__description`}>
        {tooltip ? (
          <TooltipWrapper tipContent={tooltip}>{title}</TooltipWrapper>
        ) : (
          title
        )}
      </div>
    );
  };

  const renderCard = () => {
    if (iconPosition === "left") {
      return (
        <>
          <div className={`${baseClass}__icon-count`}>
            {renderIcon()}
            {renderCount()}
          </div>
          {renderDescription()}
        </>
      );
    }

    return (
      <>
        {renderIcon()}
        {renderCount()}
        {renderDescription()}
      </>
    );
  };

  return (
    <div className={baseClass} data-testid="card">
      <Card
        className={classes}
        borderRadiusSize="large"
        path={notSupported ? undefined : path}
      >
        {renderCard()}
      </Card>
    </div>
  );
};

export default HostCountCard;
