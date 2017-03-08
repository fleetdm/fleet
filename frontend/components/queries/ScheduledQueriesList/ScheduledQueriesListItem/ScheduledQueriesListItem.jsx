import React, { Component, PropTypes } from 'react';

import Checkbox from 'components/forms/fields/Checkbox';
import ClickableTableRow from 'components/ClickableTableRow';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import { isEmpty, isEqual } from 'lodash';
import scheduledQueryInterface from 'interfaces/scheduled_query';

const baseClass = 'scheduled-query-list-item';

class ScheduledQueriesListItem extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    disabled: PropTypes.bool,
    onCheck: PropTypes.func.isRequired,
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
    const { onCheck, scheduledQuery } = this.props;

    return onCheck(value, scheduledQuery.id);
  }

  onSelect = () => {
    const { onSelect, scheduledQuery } = this.props;

    return onSelect(scheduledQuery);
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

  renderPlatformIcon = () => {
    const { scheduledQuery: { platform } } = this.props;
    const platformArr = platform ? platform.split(',') : [];

    if (isEmpty(platformArr) || platformArr.includes('all')) {
      return <PlatformIcon name="" className={`${baseClass}__icon`} />;
    }

    return platformArr.map((pltf, idx) => <PlatformIcon name={pltf} className={`${baseClass}__icon`} key={`${idx}-${pltf}`} />);
  }

  render () {
    const { checked, disabled, scheduledQuery } = this.props;
    const { id, name, interval, shard, version } = scheduledQuery;
    const { loggingTypeString, onCheck, onSelect, renderPlatformIcon } = this;

    return (
      <ClickableTableRow onClick={onSelect} className={baseClass}>
        <td>
          <Checkbox
            disabled={disabled}
            name={`scheduled-query-checkbox-${id}`}
            onChange={onCheck}
            value={checked}
          />
        </td>
        <td className="scheduled-queries-list__query-name">{name}</td>
        <td>{interval}</td>
        <td>{renderPlatformIcon()}</td>
        <td>{version ? `${version}+` : 'Any'}</td>
        <td>{shard}</td>
        <td><Icon name={loggingTypeString()} /></td>
      </ClickableTableRow>
    );
  }
}

export default ScheduledQueriesListItem;

