import React from "react";

import strUtils from "utilities/strings";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

const pluralizeHost = (count: number) => {
  return strUtils.pluralize(count, "host");
};

const baseClass = "live-results-heading";

interface IFinishButtonsProps {
  onClickDone: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickRunAgain: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

const FinishedButtons = ({
  onClickDone,
  onClickRunAgain,
}: IFinishButtonsProps) => (
  <div className={`${baseClass}__btn-wrapper`}>
    <Button className={`${baseClass}__done-btn`} onClick={onClickDone}>
      Done
    </Button>
    <Button
      className={`${baseClass}__run-btn`}
      onClick={onClickRunAgain}
      variant="success"
    >
      Run again
    </Button>
  </div>
);

interface IStopButtonProps {
  onClickStop: (evt: React.MouseEvent<HTMLButtonElement>) => void;
}

const StopButton = ({ onClickStop }: IStopButtonProps) => (
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

interface ILiveResultsHeadingProps {
  numHostsTargeted: number;
  numHostsResponded: number;
  numHostsRespondedResults: number;
  numHostsRespondedNoErrorsAndNoResults: number;
  numHostsRespondedErrors: number;
  isFinished: boolean;
  onClickDone: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickRunAgain: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickStop: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  /** Whether this is a live run of a policy or a query */
  resultsType?: "query" | "policy";
}

const LiveResultsHeading = ({
  numHostsTargeted,
  numHostsResponded,
  numHostsRespondedResults,
  numHostsRespondedNoErrorsAndNoResults,
  numHostsRespondedErrors,
  isFinished,
  onClickDone,
  onClickRunAgain,
  onClickStop,
  resultsType = "query",
}: ILiveResultsHeadingProps) => {
  const percentResponded =
    numHostsTargeted > 0
      ? Math.round((numHostsResponded / numHostsTargeted) * 100)
      : 0;

  const PAGE_TITLES = {
    RUNNING: `Running ${resultsType}`,
    FINISHED: `${resultsType[0].toUpperCase()}${resultsType.slice(1)} finished`,
  };

  const pageTitle = isFinished ? PAGE_TITLES.FINISHED : PAGE_TITLES.RUNNING;

  return (
    <div className={`${baseClass}`}>
      <h1>{pageTitle}</h1>
      <div className={`${baseClass}__information`}>
        <div className={`${baseClass}__targeted-wrapper`}>
          <span className={`${baseClass}__targeted-count`}>
            {numHostsTargeted.toLocaleString()}
          </span>
          <span>&nbsp;{pluralizeHost(numHostsTargeted)} targeted</span>
        </div>
        <div className={`${baseClass}__percent-responded`}>
          {!isFinished && <span>Fleet is talking to your hosts.&nbsp;</span>}
          <span>
            ({`${percentResponded}% `}
            <TooltipWrapper
              tipContent={
                isFinished ? (
                  <>
                    Results:{" "}
                    <b>
                      {numHostsRespondedResults}{" "}
                      {pluralizeHost(numHostsRespondedResults)}
                    </b>
                    <br />
                    No results:{" "}
                    <b>
                      {numHostsRespondedNoErrorsAndNoResults}{" "}
                      {pluralizeHost(numHostsRespondedNoErrorsAndNoResults)}
                    </b>
                    <br />
                    Errors:{" "}
                    <b>
                      {numHostsRespondedErrors}{" "}
                      {pluralizeHost(numHostsRespondedErrors)}
                    </b>
                  </>
                ) : (
                  <>
                    Hosts that respond may
                    <br /> return results, errors, or <br />
                    no results
                  </>
                )
              }
            >
              responded
            </TooltipWrapper>
            )
          </span>
          {!isFinished && (
            <Spinner
              size="x-small"
              centered={false}
              includeContainer={false}
              className={`${baseClass}__responding-spinner`}
            />
          )}
        </div>
        {!isFinished && (
          <div className={`${baseClass}__tooltip`}>
            <TooltipWrapper
              tipContent={
                <>
                  The hosts&apos; distributed interval can <br />
                  impact live query response times.
                </>
              }
            >
              Taking longer than 15 seconds?
            </TooltipWrapper>
          </div>
        )}
      </div>
      {isFinished ? (
        <FinishedButtons
          onClickDone={onClickDone}
          onClickRunAgain={onClickRunAgain}
        />
      ) : (
        <StopButton onClickStop={onClickStop} />
      )}
    </div>
  );
};

export default LiveResultsHeading;
