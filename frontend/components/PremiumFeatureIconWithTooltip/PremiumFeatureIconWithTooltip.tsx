import React from "react";
import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

const PremiumFeatureIconWithTooltip = () => {
  const tipId = uniqueId();
  return (
    <span className="premium-icon-tip">
      <span data-tip data-for={tipId}>
        <Icon name="premium-feature" />
      </span>
      <ReactTooltip
        place="top"
        type="dark"
        effect="solid"
        id={tipId}
        backgroundColor="#515774"
        delayHide={100}
        delayUpdate={500}
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
