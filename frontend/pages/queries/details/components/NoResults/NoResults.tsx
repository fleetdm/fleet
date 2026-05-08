import React from "react";

import { add, differenceInSeconds, formatDistance } from "date-fns";

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
        Fleet is collecting report results. <br />
        Check back in about {readableCheckbackTime}.
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
                The following setting prevents saving this report&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • Reports are globally disabled in organization settings.
              </div>
            </>
          );
        }
        if (discardDataEnabled) {
          return (
            <>
              <div>
                The following setting prevents saving this report&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • This report has <b>Discard data</b> enabled.
              </div>
            </>
          );
        }
        if (!loggingSnapshot) {
          return (
            <>
              <div>
                The following setting prevents saving this report&apos;s results
                in Fleet:
              </div>
              <div>
                &nbsp; • The logging setting for this report is not{" "}
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
          Results from this report are{" "}
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
                  run a{" "}
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
