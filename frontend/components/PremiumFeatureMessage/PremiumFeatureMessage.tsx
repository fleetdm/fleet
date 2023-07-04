import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

interface IPremiumFeatureMessage {
  className?: string;
}

const PremiumFeatureMessage = ({ className }: IPremiumFeatureMessage) => {
  const classes = classnames("premium-feature-message-container", className);

  return (
    <div className={classes}>
      <div className="premium-feature-message">
        <Icon name="premium-feature" />
        <p>This feature is included in Fleet Premium.</p>
        <div className="external-link-and-icon">
          <CustomLink
            url="https://fleetdm.com/upgrade"
            text="Learn more"
            newTab
          />
        </div>
      </div>
    </div>
  );
};

export default PremiumFeatureMessage;
