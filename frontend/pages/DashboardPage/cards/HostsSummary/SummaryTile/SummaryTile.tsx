import React from "react";
import { Link } from "react-router";
import { kebabCase } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import classnames from "classnames";
import { Colors } from "styles/var/colors";
import TooltipWrapper from "components/TooltipWrapper";

interface ISummaryTileProps {
  count: number;
  isLoading: boolean;
  showUI: boolean;
  title: string;
  iconName: IconNames;
  circledIcon?: boolean;
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
  circledIcon,
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

  const classes = classnames(`${baseClass}__tile`, `${kebabCase(title)}-tile`, {
    [`${baseClass}__not-supported`]: notSupported,
  });
  const tile = (
    <>
      <div className={circledIcon ? `${baseClass}__circled-icon` : ""}>
        <Icon
          name={iconName}
          size="large"
          color={iconColor}
          className={`${baseClass}__tile-icon`}
        />
      </div>
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
          {tooltip ? (
            <TooltipWrapper tipContent={tooltip}>{title}</TooltipWrapper>
          ) : (
            title
          )}
          {isSandboxMode && sandboxPremiumOnlyIcon && (
            <PremiumFeatureIconWithTooltip
              tooltipPositionOverrides={{ leftAdj: 2, topAdj: 5 }}
            />
          )}
        </div>
      </div>
    </>
  );

  // Uses Link instead of Button to include right click functionality
  // Cannot use Link disable option as it doesn't allow hover of tooltip
  return (
    <div className={baseClass} style={opacity} data-testid="tile">
      {notSupported ? (
        <div className={classes}>{tile}</div>
      ) : (
        <Link className={classes} to={path}>
          {tile}
        </Link>
      )}
    </div>
  );
};

export default SummaryTile;
