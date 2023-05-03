import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import { uniqueId } from "lodash";
import React from "react";
import ReactTooltip, { Place } from "react-tooltip";
import { COLORS } from "styles/var/colors";

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
        <Icon name="premium-feature" className="premium-feature-icon" />
      </span>
      <ReactTooltip
        place={tooltipPlace ?? "top"}
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor={COLORS["tooltip-bg"]}
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
        <CustomLink
          url="https://fleetdm.com/upgrade"
          text="Learn more"
          newTab
          multiline={false}
          iconColor="core-fleet-white"
        />
      </ReactTooltip>
    </span>
  );
};

export default PremiumFeatureIconWithTooltip;
