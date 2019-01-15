import React, { Component } from 'react';
import PropTypes from 'prop-types';

import campaignInterface from 'interfaces/campaign';
import QueryProgressDetails from 'components/queries/QueryProgressDetails';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import targetInterface from 'interfaces/target';

const baseClass = 'query-page-select-targets';

class QueryPageSelectTargets extends Component {
  static propTypes = {
    campaign: campaignInterface,
    error: PropTypes.string,
    onFetchTargets: PropTypes.func.isRequired,
    onRunQuery: PropTypes.func.isRequired,
    onStopQuery: PropTypes.func.isRequired,
    onTargetSelect: PropTypes.func.isRequired,
    queryIsRunning: PropTypes.bool,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
    queryTimerMilliseconds: PropTypes.number,
  };

  render () {
    const {
      error,
      onFetchTargets,
      onTargetSelect,
      selectedTargets,
      targetsCount,
      campaign,
      onRunQuery,
      onStopQuery,
      queryIsRunning,
      queryTimerMilliseconds,
    } = this.props;

    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <QueryProgressDetails
          campaign={campaign}
          onRunQuery={onRunQuery}
          onStopQuery={onStopQuery}
          queryIsRunning={queryIsRunning}
          queryTimerMilliseconds={queryTimerMilliseconds}
        />
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
