import React from "react";

import strUtils from "utilities/strings";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

const pluralizeHost = (count: number) => {
  return strUtils.pluralize(count, "host");
};

const baseClass = "query-results-heading";

interface IFinishButtonsProps {
  onClickDone: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickRunAgain: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

const FinishedButtons = ({
  onClickDone,
  onClickRunAgain,
}: IFinishButtonsProps) => (
  <div className={`${baseClass}__btn-wrapper`}>
    <Button
      className={`${baseClass}__done-btn`}
      onClick={onClickDone}
      variant="brand"
    >
      Done
    </Button>
    <Button
      className={`${baseClass}__run-btn`}
      onClick={onClickRunAgain}
      variant="blue-green"
    >
      Run again
    </Button>
  </div>
);

interface IStopQueryButtonProps {
  onClickStop: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

const StopQueryButton = ({ onClickStop }: IStopQueryButtonProps) => (
  <div className={`${baseClass}__btn-wrapper`}>
    <Button
      className={`${baseClass}__stop-btn`}
      onClick={onClickStop}
      variant="alert"
    >
      <>Stop</>
    </Button>
  </div>
);

interface IQueryResultsHeadingProps {
  respondedHosts: number;
  targetsTotalCount: number;
  isQueryFinished: boolean;
  onClickDone: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickRunAgain: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickStop: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

const QuertResultsHeading = ({
  respondedHosts,
  targetsTotalCount,
  isQueryFinished,
  onClickDone,
  onClickRunAgain,
  onClickStop,
}: IQueryResultsHeadingProps) => {
  const percentResponded =
    targetsTotalCount > 0
      ? Math.round((respondedHosts / targetsTotalCount) * 100)
      : 0;

  const PAGE_TITLES = {
    RUNNING: `Querying selected ${pluralizeHost(targetsTotalCount)}`,
    FINISHED: "Query finished",
  };

  const pageTitle = isQueryFinished
    ? PAGE_TITLES.FINISHED
    : PAGE_TITLES.RUNNING;

  return (
    <div className={`${baseClass}`}>
      <h1>{pageTitle}</h1>
      <div className={`${baseClass}__query-information`}>
        <div className={`${baseClass}__targeted-wrapper`}>
          <span className={`${baseClass}__targeted-count`}>
            {targetsTotalCount.toLocaleString()}
          </span>
          <span>&nbsp;{pluralizeHost(targetsTotalCount)} targeted</span>
        </div>
        <div className={`${baseClass}__percent-responded`}>
          {!isQueryFinished && (
            <span>Fleet is talking to your hosts,&nbsp;</span>
          )}
          <span>
            ({`${percentResponded}% `}
            <TooltipWrapper
              tipContent={
                <>
                  Hosts that respond may
                  <br /> return results, errors, or <br />
                  no results
                </>
              }
            >
              responded
            </TooltipWrapper>
            )
          </span>
          {!isQueryFinished && (
            <Spinner
              size="x-small"
              centered={false}
              includeContainer={false}
              className={`${baseClass}__responding-spinner`}
            />
          )}
        </div>
        {!isQueryFinished && (
          <div className={`${baseClass}__tooltip`}>
            <TooltipWrapper
              tipContent={
                <>
                  The hosts’ distributed interval can <br />
                  impact live query response times.
                </>
              }
            >
              Taking longer than 15 seconds?
            </TooltipWrapper>
          </div>
        )}
      </div>
      {isQueryFinished ? (
        <FinishedButtons
          onClickDone={onClickDone}
          onClickRunAgain={onClickRunAgain}
        />
      ) : (
        <StopQueryButton onClickStop={onClickStop} />
      )}
    </div>
  );
};

export default QuertResultsHeading;
