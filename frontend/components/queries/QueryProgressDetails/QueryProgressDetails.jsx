import React from 'react';
import PropTypes from 'prop-types';
import { get } from 'lodash';

import Button from 'components/buttons/Button';
import campaignInterface from 'interfaces/campaign';
import ProgressBar from 'components/loaders/ProgressBar';
import Timer from 'components/loaders/Timer';

const baseClass = 'query-progress-details';

const QueryProgressDetails = ({ campaign, className, onRunQuery, onStopQuery, queryIsRunning, queryTimerMilliseconds, disableRun }) => {
  const { hosts_count: hostsCount } = campaign;
  const { Metrics: metrics } = campaign;
  const { errors } = campaign;
  const totalHostsCount = get(campaign, ['totals', 'count'], 0);
  const totalRowsCount = get(campaign, ['query_results', 'length'], 0);

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

  if (!hostsCount.total) {
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
          <span className={`${baseClass}__text-online`}>Online - {metrics.OnlineHosts} hosts returning {totalRowsCount} results</span>
          <span className={`${baseClass}__text-offline`}>Offline - {metrics.OfflineHosts} hosts returning 0 results</span>
          <span className={`${baseClass}__text-error`}>Failed - {hostsCount.failed} hosts returning {errors ? errors.length : '0'} errors</span>
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
