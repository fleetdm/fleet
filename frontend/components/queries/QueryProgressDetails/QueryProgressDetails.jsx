import React from "react";
import PropTypes from "prop-types";
import { get } from "lodash";

import Button from "components/buttons/Button";
import campaignInterface from "interfaces/campaign";
import ProgressBar from "components/loaders/ProgressBar";
import Timer from "components/loaders/Timer";

const baseClass = "query-progress-details";

const QueryProgressDetails = ({
  campaign,
  className,
  onRunQuery,
  onStopQuery,
  queryIsRunning,
  queryTimerMilliseconds,
  disableRun,
}) => {
  const { hosts_count: hostsCount } = campaign;
  const { errors } = campaign;

  const totalHostsOnline = get(campaign, ["totals", "online"], 0);
  const totalHostsOffline = get(campaign, ["totals", "offline"], 0);
  const totalHostsCount = get(campaign, ["totals", "count"], 0);
  const totalRowsCount = get(campaign, ["query_results", "length"], 0);
  const campaignIsEmpty = !hostsCount.successful && hostsCount.successful !== 0;

  const onlineHostsTotalDisplay =
    totalHostsOnline === 1
      ? "1 online host"
      : `${totalHostsOnline} online hosts`;
  const onlineResultsTotalDisplay =
    totalRowsCount === 1 ? "1 result" : `${totalRowsCount} results`;
  const offlineHostsTotalDisplay =
    totalHostsOffline === 1
      ? "1 offline host"
      : `${totalHostsOffline} offline hosts`;
  const failedHostsTotalDisplay =
    hostsCount.failed === 1
      ? "1 failed host"
      : `${hostsCount.failed} failed hosts`;
  let totalErrorsDisplay = "0 errors";
  if (errors) {
    totalErrorsDisplay =
      errors.length === 1 ? "1 error" : `${errors.length} errors`;
  }

  const runQueryBtn = (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__run-btn`}
        onClick={onRunQuery}
        variant="blue-green"
        disabled={disableRun}
      >
        Run
      </Button>
    </div>
  );

  const stopQueryBtn = (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__stop-btn`}
        onClick={onStopQuery}
        variant="alert"
      >
        Stop
      </Button>
    </div>
  );

  if (!hostsCount.total && campaignIsEmpty) {
    return (
      <div className={`${baseClass} ${className}`}>
        <div className={`${baseClass}__wrapper`} />
        {queryIsRunning ? stopQueryBtn : runQueryBtn}
      </div>
    );
  }

  return (
    <div className={`${baseClass} ${className}`}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__text-wrapper`}>
          <span className={`${baseClass}__text-online`}>
            {hostsCount.successful} out of {onlineHostsTotalDisplay} responding
            <span className={`${baseClass}__text-results`}>
              {" "}
              returning {onlineResultsTotalDisplay}
            </span>
          </span>
          <span className={`${baseClass}__text-offline`}>
            {offlineHostsTotalDisplay}
            <span className={`${baseClass}__text-results`}>
              {" "}
              returning 0 results
            </span>
          </span>
          <span className={`${baseClass}__text-error`}>
            {failedHostsTotalDisplay}
            <span className={`${baseClass}__text-results`}>
              {" "}
              returning {totalErrorsDisplay}
            </span>
          </span>
        </div>
        <ProgressBar
          error={hostsCount.failed}
          max={totalHostsCount}
          success={hostsCount.successful}
        />
        {queryIsRunning && <Timer totalMilliseconds={queryTimerMilliseconds} />}
      </div>
      {queryIsRunning ? stopQueryBtn : runQueryBtn}
    </div>
  );
};

QueryProgressDetails.propTypes = {
  campaign: campaignInterface,
  className: PropTypes.string,
  onRunQuery: PropTypes.func.isRequired,
  onStopQuery: PropTypes.func.isRequired,
  queryIsRunning: PropTypes.bool,
  queryTimerMilliseconds: PropTypes.number,
  disableRun: PropTypes.bool,
};

export default QueryProgressDetails;
