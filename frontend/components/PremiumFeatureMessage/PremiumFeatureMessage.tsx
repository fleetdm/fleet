import classnames from "classnames";
import React from "react";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

interface IPremiumFeatureMessage {
  className?: string;
  /** "default" renders Option B: empty-state style with ghost table (pages/sections).
   *  "compact" renders Option A: left-aligned bordered card (modals). */
  variant?: "default" | "compact";
}

const baseClass = "premium-feature-message-container";

// Deterministic skeleton widths matching Figma Option B wireframe
const GHOST_SKELETON_WIDTHS = [294, 272, 302, 276, 298, 268];

const PremiumFeatureMessage = ({
  className,
  variant = "default",
}: IPremiumFeatureMessage) => {
  const isCompact = variant === "compact";
  const classes = classnames(baseClass, className, {
    [`${baseClass}--compact`]: isCompact,
  });

  if (isCompact) {
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
  }

  return (
    <div className={classes}>
      <div className={`${baseClass}__ghost-table`} aria-hidden="true">
        <div className={`${baseClass}__ghost-header`}>
          <div
            className={`${baseClass}__ghost-skeleton`}
            style={{ width: 80 }}
          />
        </div>
        {GHOST_SKELETON_WIDTHS.map((width) => (
          <div key={width} className={`${baseClass}__ghost-cell`}>
            <div className={`${baseClass}__ghost-skeleton`} style={{ width }} />
          </div>
        ))}
      </div>
      <div className={`${baseClass}__content`}>
        <Icon name="premium-feature" />
        <p className={`${baseClass}__title`}>Included in Fleet Premium</p>
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
