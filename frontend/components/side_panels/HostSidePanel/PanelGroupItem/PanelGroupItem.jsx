import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Icon from 'components/Icon';
import { iconClassForLabel } from './helpers';

const baseClass = 'panel-group-item';

class PanelGroupItem extends Component {
  static propTypes = {
    item: PropTypes.shape({
      hosts_count: PropTypes.number,
      title: PropTypes.string,
      type: PropTypes.string,
    }).isRequired,
    onLabelClick: PropTypes.func,
    isSelected: PropTypes.bool,
  };

  render () {
    const { item, onLabelClick, isSelected } = this.props;
    const {
      count,
      display_text: displayText,
    } = item;
    const wrapperClassName = classnames(baseClass, `${baseClass}__wrapper`, {
      [`${baseClass}__wrapper--is-selected`]: isSelected,
    });

    return (
      <button className={`${wrapperClassName} button button--unstyled`} onClick={onLabelClick}>
        <Icon name={iconClassForLabel(item)} />
        <span>{displayText}</span>
        <span>{count}</span>
      </button>
    );
  }
}

export default PanelGroupItem;
