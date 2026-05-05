import React from "react";

import CustomLink from "components/CustomLink";

const baseClass = "learn-fleet";

const LearnFleet = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <p>
        Want to explore Fleet&apos;s features? Learn how to ask questions about
        your device using reports.
      </p>
      <CustomLink
        className={`${baseClass}__action-button`}
        url="https://fleetdm.com/docs/using-fleet/learn-how-to-use-fleet"
        text="Learn how to use Fleet"
        newTab
      />
    </div>
  );
};

export default LearnFleet;
