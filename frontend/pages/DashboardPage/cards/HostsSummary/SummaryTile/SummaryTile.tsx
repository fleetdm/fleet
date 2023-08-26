import React from "react";
import { browserHistory } from "react-router";
import Button from "components/buttons/Button";
import { kebabCase } from "lodash";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import classnames from "classnames";
import { Colors } from "styles/var/colors";
import NewTooltipWrapper from "components/NewTooltipWrapper";

interface ISummaryTileProps {
  count: number;
  isLoading: boolean;
  showUI: boolean;
  title: string;
  iconName: IconNames;
  iconColor?: Colors;
  path: string;
  tooltip?: string;
  isSandboxMode?: boolean;
  sandboxPremiumOnlyIcon?: boolean;
  notSupported?: boolean;
}

const baseClass = "summary-tile";

const SummaryTile = ({
  count,
  isLoading,
  showUI, // false on first load only
  title,
  iconName,
  iconColor,
  path,
  tooltip,
  isSandboxMode = false,
  sandboxPremiumOnlyIcon = false,
  notSupported = false,
}: ISummaryTileProps): JSX.Element => {
  const numberWithCommas = (x: number): string => {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  };
  // Renders opaque information as host information is loading
  let opacity = { opacity: 0 };
  if (showUI) {
    opacity = isLoading ? { opacity: 0.4 } : { opacity: 1 };
  }

  const handleClick = () => {
    browserHistory.push(path);
  };

  const classes = classnames(`${baseClass}__tile`, `${kebabCase(title)}-tile`, {
    [`${baseClass}__not-supported`]: notSupported,
  });
  // TODO CHANGE THIS BACK!
  const getWrappedTitle = () => {
    if (tooltip) {
      if (title === "Missing hosts") {
        return <TooltipWrapper tipContent={tooltip}>{title}</TooltipWrapper>;
      }
      return (
        <NewTooltipWrapper tipContent={tooltip}>{title}</NewTooltipWrapper>
      );
    }
    return title;
  };
  return (
    <div className={baseClass} style={opacity} data-testid="tile">
      <Button
        className={classes}
        variant="unstyled"
        onClick={() => handleClick()}
        disabled={notSupported}
      >
        <>
          <Icon
            name={iconName}
            color={iconColor}
            className={`${baseClass}__tile-icon`}
          />
          <div>
            {notSupported ? (
              <div className={`${baseClass}__not-supported-text`}>
                Not supported
              </div>
            ) : (
              <div
                className={`${baseClass}__count ${baseClass}__count--${kebabCase(
                  title
                )}`}
              >
                {numberWithCommas(count)}
              </div>
            )}
            <div className={`${baseClass}__description`}>
              {getWrappedTitle()}
              {isSandboxMode && sandboxPremiumOnlyIcon && (
                <PremiumFeatureIconWithTooltip
                  tooltipPositionOverrides={{ leftAdj: 2, topAdj: 5 }}
                />
              )}
            </div>
          </div>
        </>
      </Button>
    </div>
  );
};

export default SummaryTile;
