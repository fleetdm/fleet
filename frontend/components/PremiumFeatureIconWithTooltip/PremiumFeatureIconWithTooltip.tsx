import Icon from "components/Icon";
import { uniqueId } from "lodash";
import React from "react";
import ReactTooltip, { Place } from "react-tooltip";

interface IPremiumFeatureIconWithTooltip {
  tooltipPlace?: Place;
  tooltipDelayHide?: number;
  tooltipPositionOverrides?: {
    leftAdj?: number;
    topAdj?: number;
  };
}
const PremiumFeatureIconWithTooltip = ({
  tooltipPlace,
  tooltipDelayHide = 100,
  tooltipPositionOverrides,
}: IPremiumFeatureIconWithTooltip) => {
  const [leftAdj, topAdj] = [
    tooltipPositionOverrides?.leftAdj ?? 0,
    tooltipPositionOverrides?.topAdj ?? 0,
  ];
  const tipId = uniqueId();
  return (
    <span className="premium-icon-tip">
      <span data-tip data-for={tipId}>
        <Icon name="premium-feature" />
      </span>
      <ReactTooltip
        place={tooltipPlace ?? "top"}
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor="#515774"
        delayHide={tooltipDelayHide}
        delayUpdate={500}
        overridePosition={(pos: { left: number; top: number }) => {
          return {
            left: pos.left + leftAdj,
            top: pos.top + topAdj,
          };
        }}
      >
        {`This is a Fleet Premium feature. `}
        <a href="https://fleetdm.com/upgrade" rel="noreferrer" target="_blank">
          {"Learn more"}
        </a>
        .
      </ReactTooltip>
    </span>
  );
};

export default PremiumFeatureIconWithTooltip;
