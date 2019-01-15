import React from 'react';
import PropTypes from 'prop-types';
import { get } from 'lodash';

import Button from 'components/buttons/Button';
import campaignInterface from 'interfaces/campaign';
import ProgressBar from 'components/loaders/ProgressBar';
import Timer from 'components/loaders/Timer';

const baseClass = 'query-progress-details';

const QueryProgressDetails = ({ campaign, className, onRunQuery, onStopQuery, queryIsRunning, queryTimerMilliseconds }) => {
  const { hosts_count: hostsCount } = campaign;
  const totalHostsCount = get(campaign, ['totals', 'count'], 0);
  const totalRowsCount = get(campaign, ['query_results', 'length'], 0);

  const runQueryBtn = (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__run-btn`}
        onClick={onRunQuery}
        variant="success"
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
        <span>
          <b>{hostsCount.total}</b>&nbsp;of&nbsp;
          <b>{totalHostsCount} Hosts</b>&nbsp;Returning&nbsp;
          <b>{totalRowsCount} Records&nbsp;</b>
          <em>({hostsCount.failed} failed)</em>
        </span>
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
};

export default QueryProgressDetails;
