import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import React from "react";

const PremiumFeatureMessage = () => {
  return (
    <div className="premium-feature-message-container">
      <div className="premium-feature-message">
        <Icon name="premium-feature" />
        <p>This feature is included in Fleet Premium.</p>
        <div className="external-link-and-icon">
          <CustomLink url="https://fleetdm.com/upgrade" text="Learn more" />
          <Icon name="external-link" />
        </div>
      </div>
    </div>
  );
};

export default PremiumFeatureMessage;
