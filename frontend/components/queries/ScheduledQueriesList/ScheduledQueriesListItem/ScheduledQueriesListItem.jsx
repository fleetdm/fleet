import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Checkbox from 'components/forms/fields/Checkbox';
import ClickableTableRow from 'components/ClickableTableRow';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import { includes, isEmpty, isEqual } from 'lodash';
import scheduledQueryInterface from 'interfaces/scheduled_query';

const baseClass = 'scheduled-query-list-item';

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

    if (removed) {
      return 'plus-minus';
    }

    return 'bold-plus';
  }

  renderPlatformIcon = () => {
    const { scheduledQuery: { platform } } = this.props;
    const platformArr = platform ? platform.split(',') : [];

    if (isEmpty(platformArr) || includes(platformArr, 'all')) {
      return <PlatformIcon name="all" title="All Platforms" className={`${baseClass}__icon`} />;
    }

    return platformArr.map((pltf, idx) => <PlatformIcon name={pltf} title={pltf} className={`${baseClass}__icon`} key={`${idx}-${pltf}`} />);
  }

  render () {
    const { checked, disabled, isSelected, scheduledQuery } = this.props;
    const { id, name, interval, shard, version } = scheduledQuery;
    const { loggingTypeString, onDblClick, onCheck, onSelect, renderPlatformIcon } = this;
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
        <td>{renderPlatformIcon()}</td>
        <td>{version ? `${version}+` : 'Any'}</td>
        <td>{shard}</td>
        <td><Icon name={loggingTypeString()} /></td>
      </ClickableTableRow>
    );
  }
}

export default ScheduledQueriesListItem;

