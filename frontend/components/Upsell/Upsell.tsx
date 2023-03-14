import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import React from "react";

const Upsell = () => {
  return (
    <div className="upsell-container">
      <div className="upsell">
        <p>This feature is included in Fleet Premium.</p>
        <div className="external-link">
          <CustomLink url="https://fleetdm.com/upgrade" text="Learn more" />
          <Icon name="external-link" />
        </div>
      </div>
    </div>
  );
};

export default Upsell;
