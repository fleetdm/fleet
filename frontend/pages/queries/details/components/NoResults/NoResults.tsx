import React from "react";

import { add, differenceInSeconds, formatDistanceStrict } from "date-fns";

import PATHS from "router/paths";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import EmptyState from "components/EmptyState";
import CustomLink from "components/CustomLink";

interface INoResultsProps {
  queryId: number;
  queryInterval?: number;
  queryUpdatedAt?: string;
  disabledCaching: boolean;
  disabledCachingGlobally: boolean;
  discardDataEnabled: boolean;
  loggingSnapshot: boolean;
  canLiveQuery?: boolean;
  canEditQuery?: boolean;
}

const baseClass = "no-results";

const NoResults = ({
  queryId,
  queryInterval,
  queryUpdatedAt,
  disabledCaching,
  disabledCachingGlobally,
  discardDataEnabled,
  loggingSnapshot,
  canLiveQuery,
  canEditQuery,
}: INoResultsProps): JSX.Element => {
  // Give up on "still collecting" once more than one interval (plus a
  // buffer for config propagation) has passed since the report was last
  // saved — after that, something other than "waiting for the next
  // checkpoint" is more likely going on.
  const secondsSinceUpdate = queryUpdatedAt
    ? differenceInSeconds(new Date(), new Date(queryUpdatedAt))
    : 0;
  const collectingResults =
    (queryInterval ?? 0) > 0 && secondsSinceUpdate < (queryInterval || 0) + 60;

  // Fleet's scheduled reports fire on a fixed wall-clock grid (epoch time
  // that's a multiple of the interval), not counted from when the report
  // was saved, so estimate the next checkpoint directly instead of
  // assuming a full interval from save time. Hosts also need up to ~60s
  // (config_tls_refresh) to pick up a newly saved/edited schedule before a
  // checkpoint can apply.
  const secondsUntilNextCheckpoint = () => {
    if (!queryInterval) {
      return 0;
    }
    const nowSeconds = Date.now() / 1000;
    const updatedAtSeconds = queryUpdatedAt
      ? new Date(queryUpdatedAt).getTime() / 1000
      : nowSeconds;
    const earliestApplicable = Math.max(nowSeconds, updatedAtSeconds + 60);
    const nextCheckpoint =
      Math.ceil(earliestApplicable / queryInterval) * queryInterval;
    return nextCheckpoint - nowSeconds;
  };

  // Converts seconds until the next checkpoint to human readable format
  const readableCheckbackTime = formatDistanceStrict(
    add(new Date(), { seconds: secondsUntilNextCheckpoint() }),
    new Date()
  );

  // Collecting results state only shows if caching is enabled
  if (collectingResults && !disabledCaching) {
    const collectingResultsInfo = () => (
      <>
        Results expected in about {readableCheckbackTime} if hosts are online
        then.{" "}
        <CustomLink
          url="https://fleetdm.com/guides/reports#schedule-a-report"
          text="Learn more"
          newTab
        />
      </>
    );

    return (
      <EmptyState
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
                <b>Store report results</b> is globally disabled in organization
                settings.
              </div>
            </>
          );
        }
        if (discardDataEnabled) {
          return (
            <>
              <div>
                <b>Store data</b> is disabled.
              </div>
            </>
          );
        }
        if (!loggingSnapshot) {
          return (
            <>
              <div>
                <b>Differential logging</b> is enabled.
              </div>
            </>
          );
        }
        return "Unknown";
      };
      return [
        "Nothing to report",
        <>
          Results from this report are{" "}
          <TooltipWrapper tipContent={tipContent()}>
            not stored in Fleet
          </TooltipWrapper>
          .
        </>,
      ];
    }
    if (!queryInterval) {
      return [
        "Nothing to report",
        <>
          This report does not collect data on a schedule.
          {(canEditQuery || canLiveQuery) && (
            <>
              <br />
              {canEditQuery && (
                <>
                  Add an <strong>interval</strong>
                </>
              )}
              {canEditQuery && canLiveQuery && " or "}
              {canLiveQuery && (
                <>
                  {canEditQuery ? "run" : "Run"} a{" "}
                  <CustomLink
                    url={PATHS.LIVE_REPORT(queryId)}
                    text="live report"
                  />
                </>
              )}{" "}
              to see results.
            </>
          )}
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
        This report has returned no data so far.
        {canLiveQuery && (
          <>
            <br />
            Expecting to see results? Run a{" "}
            <CustomLink
              url={PATHS.LIVE_REPORT(queryId)}
              text="live report"
            />{" "}
            to troubleshoot.
          </>
        )}
      </>,
    ];
  };

  const [emptyHeader, emptyDetails] = getNoResultsInfo();
  return (
    <EmptyState
      className={baseClass}
      header={emptyHeader}
      info={emptyDetails}
    />
  );
};

export default NoResults;
