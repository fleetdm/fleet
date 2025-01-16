import React from "react";
import { Link } from "react-router";
import { kebabCase } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import classnames from "classnames";
import TooltipWrapper from "components/TooltipWrapper";

interface IHostCountCard {
  count: number;
  isLoading: boolean;
  showUI: boolean;
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
  isLoading,
  showUI, // false on first load only
  title,
  iconName,
  path,
  tooltip,
  notSupported = false,
  className,
  iconPosition = "top",
}: IHostCountCard): JSX.Element => {
  const numberWithCommas = (x: number): string => {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  };
  // Renders opaque information as host information is loading
  let opacity = { opacity: 0 };
  if (showUI) {
    opacity = isLoading ? { opacity: 0.4 } : { opacity: 1 };
  }

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
        {numberWithCommas(count)}
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

  // Uses Link instead of Button to include right click functionality
  // Cannot use Link disable option as it doesn't allow hover of tooltip
  return (
    <div className={baseClass} style={opacity} data-testid="card">
      {notSupported ? (
        <div className={classes}>{renderCard()}</div>
      ) : (
        <Link className={classes} to={path}>
          {renderCard()}
        </Link>
      )}
    </div>
  );
};

export default HostCountCard;
