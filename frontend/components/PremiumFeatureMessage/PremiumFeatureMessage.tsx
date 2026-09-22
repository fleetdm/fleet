import classnames from "classnames";
import React from "react";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

interface IPremiumFeatureMessage {
  className?: string;
  /** "default" renders a bordered card container (for pages/sections).
   *  "compact" renders a small left-aligned inline layout (for modals). */
  variant?: "default" | "compact";
}

const baseClass = "premium-feature-message-container";

const PremiumFeatureMessage = ({
  className,
  variant = "default",
}: IPremiumFeatureMessage) => {
  const classes = classnames(baseClass, className, {
    [`${baseClass}--compact`]: variant === "compact",
  });

  return (
    <div className={classes}>
      <div className="premium-feature-message">
        <Icon name="premium-feature" />
        <p>This feature is included in Fleet Premium.</p>
        <CustomLink
          url="https://fleetdm.com/upgrade"
          text="Learn more"
          newTab
        />
      </div>
    </div>
  );
};

export default PremiumFeatureMessage;
