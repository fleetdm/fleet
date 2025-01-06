import React from "react";
import LastUpdatedText from "components/LastUpdatedText";

const baseClass = "last-updated-host-count";

interface ILastUpdatedHostCount {
  hostCount?: string | number;
  lastUpdatedAt?: string;
}

const LastUpdatedHostCount = ({
  hostCount,
  lastUpdatedAt,
}: ILastUpdatedHostCount): JSX.Element => {
  const tooltipContent = (
    <>
      The last time host data was updated. <br />
      Click <b>View all hosts</b> to see the most
      <br /> up-to-date host count.
    </>
  );

  return (
    <div className={baseClass}>
      <>{hostCount}</>
      {lastUpdatedAt && (
        <LastUpdatedText
          lastUpdatedAt={lastUpdatedAt}
          customTooltipText={tooltipContent}
        />
      )}
    </div>
  );
};

export default LastUpdatedHostCount;
