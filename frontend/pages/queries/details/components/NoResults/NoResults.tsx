import React from "react";

import differenceInSeconds from "date-fns/differenceInSeconds";
import formatDistance from "date-fns/formatDistance";
import add from "date-fns/add";

import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import EmptyTable from "components/EmptyTable/EmptyTable";

interface INoResultsProps {
  queryInterval?: number;
  queryUpdatedAt?: string;
  disabledCaching: boolean;
  disabledCachingGlobally: boolean;
  discardDataEnabled: boolean;
  loggingSnapshot: boolean;
}

const baseClass = "no-results";

const NoResults = ({
  queryInterval,
  queryUpdatedAt,
  disabledCaching,
  disabledCachingGlobally,
  discardDataEnabled,
  loggingSnapshot,
}: INoResultsProps): JSX.Element => {
  // Returns how many seconds it takes to expect a cached update
  const secondsCheckbackTime = () => {
    const secondsSinceUpdate = queryUpdatedAt
      ? differenceInSeconds(new Date(), new Date(queryUpdatedAt))
      : 0;
    const secondsUpdateWaittime = (queryInterval || 0) + 60;
    return secondsUpdateWaittime - secondsSinceUpdate;
  };

  // Update status of collecting cached results
  const collectingResults = secondsCheckbackTime() > 0;

  // Converts seconds takes to update to human readable format
  const readableCheckbackTime = formatDistance(
    add(new Date(), { seconds: secondsCheckbackTime() }),
    new Date()
  );

  // Collecting results state
  if (collectingResults) {
    const collectingResultsInfo = () =>
      `Fleet is collecting query results. Check back in about ${readableCheckbackTime}.`;

    return (
      <EmptyTable
        iconName="collecting-results"
        header={"Collecting results..."}
        info={collectingResultsInfo()}
      />
    );
  }

  const noResultsInfo = () => {
    if (!queryInterval) {
      return (
        <>
          This query does not collect data on a schedule. Add a{" "}
          <strong>frequency</strong> or run this as a live query to see results.
        </>
      );
    }
    if (disabledCaching) {
      const tipContent = () => {
        if (disabledCachingGlobally) {
          return "The following setting prevents saving this query's results in Fleet:<ul><li>Query reports are globally disabled in organization settings.</li></ul>";
        }
        if (discardDataEnabled) {
          return "The following setting prevents saving this query's results in Fleet:<ul><li>This query has Discard data enabled.</li></ul>";
        }
        if (!loggingSnapshot) {
          return "The following setting prevents saving this query's results in Fleet:<ul><li>The logging setting for this query is not Snapshot.</li></ul>";
        }
        return "Unknown";
      };
      return (
        <>
          Results from this query are{" "}
          <TooltipWrapper tipContent={tipContent()}>
            not reported in Fleet
          </TooltipWrapper>
          .
        </>
      );
    }
    // No errors will be reported in V1
    // if (errorsOnly) {
    //   return (
    //     <>
    //       This query had trouble collecting data on some hosts. Check out the{" "}
    //       <strong>Errors</strong> tab to see why.
    //     </>
    //   );
    // }
    return "This query has returned no data so far.";
  };

  return (
    <EmptyTable
      className={baseClass}
      iconName="empty-software"
      header={"Nothing to report yet"}
      info={noResultsInfo()}
    />
  );
};

export default NoResults;
