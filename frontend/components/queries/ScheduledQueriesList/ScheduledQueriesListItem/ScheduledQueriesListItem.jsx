import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Checkbox from 'components/forms/fields/Checkbox';
import ClickableTableRow from 'components/ClickableTableRow';
import KolideIcon from 'components/icons/KolideIcon';
import { isEqual, find } from 'lodash';
import scheduledQueryInterface from 'interfaces/scheduled_query';

const baseClass = 'scheduled-query-list-item';

const generatePlatformText = (platforms) => {
  const ALL_PLATFORMS = [
    { text: 'All', value: 'all' },
    { text: 'macOS', value: 'darwin' },
    { text: 'Windows', value: 'windows' },
    { text: 'Linux', value: 'linux' },
  ];
  console.log(platforms)
  if (platforms) {
    const platformArray = platforms.split(',');
    console.log(platformArray);
    const textArray = platformArray.map((platform) => {
      const text = find(ALL_PLATFORMS, { value: platform }).text;

      return text;
    });

    const displayText = textArray.join(', ');

    return displayText;
  }

  return '---';
};

class ScheduledQueriesListItem extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    disabled: PropTypes.bool,
    isSelected: PropTypes.bool,
    onCheck: PropTypes.func.isRequired,
    onSelect: PropTypes.func.isRequired,
    onDblClick: PropTypes.func.isRequired,
    scheduledQuery: scheduledQueryInterface.isRequired,
  };

  shouldComponentUpdate (nextProps, nextState) {
    if (isEqual(nextProps, this.props) && isEqual(nextState, this.state)) {
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

  onDblClick = () => {
    const { onDblClick, scheduledQuery } = this.props;

    return onDblClick(scheduledQuery.query_id);
  }

  loggingTypeString = () => {
    const { scheduledQuery: { snapshot, removed } } = this.props;

    if (snapshot) {
      return 'camera';
    }

    // Default is differential with removals, so we treat null as removed = true
    if (removed !== false) {
      return 'plus-minus';
    }

    return 'bold-plus';
  }

  render () {
    const { checked, disabled, isSelected, scheduledQuery } = this.props;
    const { id, query_name: name, interval, shard, version, platform } = scheduledQuery;
    const { loggingTypeString, onDblClick, onCheck, onSelect } = this;
    const rowClassname = classnames(baseClass, {
      [`${baseClass}--selected`]: isSelected,
    });

    return (
      <ClickableTableRow onClick={onSelect} onDoubleClick={onDblClick} className={rowClassname}>
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
        <td>{generatePlatformText(platform)}</td>
        <td>{version ? `${version}+` : 'Any'}</td>
        <td>{shard}</td>
        <td><KolideIcon name={loggingTypeString()} /></td>
      </ClickableTableRow>
    );
  }
}

export default ScheduledQueriesListItem;
