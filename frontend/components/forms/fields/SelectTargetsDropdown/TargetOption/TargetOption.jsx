import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';
import targetInterface from 'interfaces/target';
import TargetIcon from './TargetIcon';

const baseClass = 'target-option';

class TargetOption extends Component {
  static propTypes = {
    onMoreInfoClick: PropTypes.func,
    onSelect: PropTypes.func,
    target: targetInterface.isRequired,
  };

  handleSelect = (evt) => {
    const { onSelect, target } = this.props;

    return onSelect(target, evt);
  }

  renderTargetDetail = () => {
    const { target } = this.props;
    const {
      count,
      host_ip_address: hostIpAddress,
      target_type: targetType,
    } = target;

    if (targetType === 'hosts') {
      if (!hostIpAddress) {
        return false;
      }

      return (
        <span>
          <span className={`${baseClass}__delimeter`}>&bull;</span>
          <span className={`${baseClass}__ip`}>{hostIpAddress}</span>
        </span>
      );
    }

    return <span className={`${baseClass}__count`}>{count} hosts</span>;
  }

  render () {
    const { onMoreInfoClick, target } = this.props;
    const { display_text: displayText, target_type: targetType } = target;
    const {
      handleSelect,
      renderTargetDetail,
    } = this;
    const wrapperClassName = classnames(`${baseClass}__wrapper`, {
      'is-label': targetType === 'labels',
      'is-host': targetType === 'hosts',
    });

    return (
      <div className={wrapperClassName}>
        <button className={`button button--unstyled ${baseClass}__target-content`} onClick={onMoreInfoClick(target)}>
          <TargetIcon target={target} />
          <span className={`${baseClass}__label-label`}>{displayText}</span>
          {renderTargetDetail()}
        </button>
        <button className={`button button--unstyled ${baseClass}__add-btn`} onClick={handleSelect}>
          <Icon name="add-button" />
        </button>
      </div>
    );
  }
}

export default TargetOption;
