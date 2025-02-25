import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

interface IPremiumFeatureMessage {
  className?: string;
  /** Aligns premium message, default: centered */
  alignment?: "left";
}

const baseClass = "premium-feature-message-container";

const PremiumFeatureMessage = ({
  className,
  alignment,
}: IPremiumFeatureMessage) => {
  const classes = classnames(
    baseClass,
    {
      [`${baseClass}__align-${alignment}`]: alignment !== undefined,
    },
    className
  );

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
