import React from "react";

import { add, differenceInSeconds, formatDistance } from "date-fns";

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
  const collectingResults =
    (queryInterval ?? 0) > 0 && secondsCheckbackTime() > 0;

  // Converts seconds takes to update to human readable format
  const readableCheckbackTime = formatDistance(
    add(new Date(), { seconds: secondsCheckbackTime() }),
    new Date()
  );

  // Collecting results state only shows if caching is enabled
  if (collectingResults && !disabledCaching) {
    const collectingResultsInfo = () => (
      <>
        Fleet is collecting query results. <br />
        Check back in about {readableCheckbackTime}.
      </>
    );

    return (
      <EmptyTable
        graphicName="collecting-results"
        header="Collecting results..."
        info={collectingResultsInfo()}
      />
    );
  }

  const getNoResultsInfo = () => {
    // In order of empty page priority
    if (disabledCaching) {
      const tipContent = () => {
        if (disabledCachingGlobally) {
          return (
            <>
              <div>
                The following setting prevents saving this query&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • Query reports are globally disabled in organization
                settings.
              </div>
            </>
          );
        }
        if (discardDataEnabled) {
          return (
            <>
              <div>
                The following setting prevents saving this query&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • This query has <b>Discard data</b> enabled.
              </div>
            </>
          );
        }
        if (!loggingSnapshot) {
          return (
            <>
              <div>
                The following setting prevents saving this query&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • The logging setting for this query is not{" "}
                <b>Snapshot</b>.
              </div>
            </>
          );
        }
        return "Unknown";
      };
      return [
        "Nothing to report",
        <>
          Results from this query are{" "}
          <TooltipWrapper tipContent={tipContent()}>
            not reported in Fleet
          </TooltipWrapper>
          .
        </>,
      ];
    }
    if (!queryInterval) {
      return [
        "Nothing to report",
        <>
          This query does not collect data on a schedule. Add <br />a{" "}
          <strong>frequency</strong> or run this as a live query to see results.
        </>,
      ];
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
    return [
      "Nothing to report yet",
      <>
        This query has returned no data so far. If you&apos;re <br />
        expecting to see results, try running a live query to
        <br />
        get diagnostics.
      </>,
    ];
  };

  const [emptyHeader, emptyDetails] = getNoResultsInfo();
  return (
    <EmptyTable
      className={baseClass}
      graphicName="empty-software"
      header={emptyHeader}
      info={emptyDetails}
    />
  );
};

export default NoResults;
