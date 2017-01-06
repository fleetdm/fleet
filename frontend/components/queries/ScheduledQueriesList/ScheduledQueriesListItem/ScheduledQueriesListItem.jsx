import React, { Component, PropTypes } from 'react';

import Checkbox from 'components/forms/fields/Checkbox';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import { isEqual } from 'lodash';
import scheduledQueryInterface from 'interfaces/scheduled_query';

class ScheduledQueriesListItem extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    disabled: PropTypes.bool,
    onSelect: PropTypes.func.isRequired,
    scheduledQuery: scheduledQueryInterface.isRequired,
  };

  shouldComponentUpdate (nextProps) {
    if (isEqual(nextProps, this.props)) {
      return false;
    }

    return true;
  }

  onCheck = (value) => {
    const { onSelect, scheduledQuery } = this.props;

    return onSelect(value, scheduledQuery.id);
  }

  loggingTypeString = () => {
    const { scheduledQuery: { snapshot, removed } } = this.props;

    if (snapshot) {
      return 'camera';
    }

    if (removed) {
      return 'plus-minus';
    }

    return 'bold-plus';
  }

  render () {
    const { checked, disabled, scheduledQuery } = this.props;
    const { onCheck } = this;
    const { id, name, interval, platform, version } = scheduledQuery;
    const { loggingTypeString } = this;

    return (
      <tr>
        <td>
          <Checkbox
            disabled={disabled}
            name={`scheduled-query-checkbox-${id}`}
            onChange={onCheck}
            value={checked}
          />
        </td>
        <td>{name}</td>
        <td>{interval}</td>
        <td><PlatformIcon name={platform} /></td>
        <td>{version}</td>
        <td><Icon name={loggingTypeString()} /></td>
      </tr>
    );
  }
}

export default ScheduledQueriesListItem;

