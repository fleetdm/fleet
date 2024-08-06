import React from "react";

import LastUpdatedText from "components/LastUpdatedText";

interface ISoftwareLastUpdatedInfoProps {
  lastUpdatedAt: string;
}

const SoftwareLastUpdatedInfo = ({
  lastUpdatedAt,
}: ISoftwareLastUpdatedInfoProps) => {
  return (
    <LastUpdatedText
      lastUpdatedAt={lastUpdatedAt}
      customTooltipText={
        <>
          The last time software data was <br />
          updated, including vulnerabilities <br />
          and host counts.
        </>
      }
    />
  );
};

export default SoftwareLastUpdatedInfo;
