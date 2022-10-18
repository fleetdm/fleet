import React from "react";
import { browserHistory } from "react-router";
import Button from "components/buttons/Button";
import { kebabCase } from "lodash";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

interface ISummaryTileProps {
  count: number;
  isLoading: boolean;
  showUI: boolean;
  title: string;
  iconName?: IconNames;
  tooltip?: string;
  path: string;
}

const baseClass = "summary-tile";

const SummaryTile = ({
  count,
  isLoading,
  showUI, // false on first load only
  title,
  iconName,
  tooltip,
  path,
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

  return (
    <div className={baseClass} style={opacity} data-testid="tile">
      <Button
        className={`${baseClass}__tile ${kebabCase(title)}-tile`}
        variant="unstyled"
        onClick={() => handleClick()}
      >
        <>
          {iconName && (
            <div className={`${baseClass}__icon-wrapper`}>
              <Icon name={iconName} className={`${baseClass}__event-icon`} />
            </div>
          )}
          <div>
            <div
              className={`${baseClass}__count ${baseClass}__count--${kebabCase(
                title
              )}`}
            >
              {numberWithCommas(count)}
            </div>
            <div className={`${baseClass}__description`}>
              {tooltip ? (
                <TooltipWrapper tipContent={tooltip}>{title}</TooltipWrapper>
              ) : (
                title
              )}
            </div>
          </div>
        </>
      </Button>
    </div>
  );
};

export default SummaryTile;
