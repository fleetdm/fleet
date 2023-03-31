import Icon from "components/Icon";
import { uniqueId } from "lodash";
import React from "react";
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
        <a href="https://fleetdm.com/upgrade" rel="noreferrer" target="_blank">
          {"Learn more"}
        </a>
        .
      </ReactTooltip>
    </span>
  );
};

export default PremiumFeatureIconWithTooltip;
