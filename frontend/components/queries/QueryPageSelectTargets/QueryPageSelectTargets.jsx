import React, { Component, PropTypes } from 'react';
import { get } from 'lodash';

import Button from 'components/buttons/Button';
import campaignInterface from 'interfaces/campaign';
import ProgressBar from 'components/loaders/ProgressBar';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import targetInterface from 'interfaces/target';
import Timer from 'components/loaders/Timer';

const baseClass = 'query-page-select-targets';

class QueryPageSelectTargets extends Component {
  static propTypes = {
    campaign: campaignInterface,
    error: PropTypes.string,
    onFetchTargets: PropTypes.func.isRequired,
    onRunQuery: PropTypes.func.isRequired,
    onStopQuery: PropTypes.func.isRequired,
    onTargetSelect: PropTypes.func.isRequired,
    query: PropTypes.string,
    queryIsRunning: PropTypes.bool,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
  };

  onRunQuery = () => {
    const { onRunQuery, query } = this.props;

    return onRunQuery(query);
  }

  renderProgressDetails = () => {
    const {
      campaign,
      onStopQuery,
      queryIsRunning,
    } = this.props;
    const { onRunQuery } = this;
    const { hosts_count: hostsCount } = campaign;
    const totalHostsCount = get(campaign, ['totals', 'count'], 0);
    const totalRowsCount = get(campaign, ['query_results', 'length'], 0);

    const runQueryBtn = (
      <div className={`${baseClass}__query-btn-wrapper`}>
        {queryIsRunning && <Timer running={queryIsRunning} />}
        <Button
          className={`${baseClass}__run-query-btn`}
          onClick={onRunQuery}
          variant="success"
        >
          Run
        </Button>
      </div>
    );
    const stopQueryBtn = (
      <div className={`${baseClass}__query-btn-wrapper`}>
        {queryIsRunning && <Timer running={queryIsRunning} />}
        <Button
          className={`${baseClass}__stop-query-btn`}
          onClick={onStopQuery}
          variant="alert"
        >
          Stop
        </Button>
      </div>
    );

    if (!hostsCount.total) {
      return (
        <div className={`${baseClass}__progress-wrapper`}>
          <div className={`${baseClass}__progress-details`} />
          {queryIsRunning ? stopQueryBtn : runQueryBtn}
        </div>
      );
    }

    return (
      <div className={`${baseClass}__progress-wrapper`}>
        <div className={`${baseClass}__progress-details`}>
          <span>
            <b>{hostsCount.total}</b>&nbsp;of&nbsp;
            <b>{totalHostsCount} Hosts</b>&nbsp;Returning&nbsp;
            <b>{totalRowsCount} Records&nbsp;</b>
            ({hostsCount.failed} failed)
          </span>
          <ProgressBar
            error={hostsCount.failed}
            max={totalHostsCount}
            success={hostsCount.successful}
          />
        </div>
        {queryIsRunning ? stopQueryBtn : runQueryBtn}
      </div>
    );
  }

  render () {
    const {
      error,
      onFetchTargets,
      onTargetSelect,
      selectedTargets,
      targetsCount,
    } = this.props;
    const { renderProgressDetails } = this;

    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        {renderProgressDetails()}
        <SelectTargetsDropdown
          error={error}
          onFetchTargets={onFetchTargets}
          onSelect={onTargetSelect}
          selectedTargets={selectedTargets}
          targetsCount={targetsCount}
          label="Select Targets"
        />
      </div>
    );
  }
}

export default QueryPageSelectTargets;
