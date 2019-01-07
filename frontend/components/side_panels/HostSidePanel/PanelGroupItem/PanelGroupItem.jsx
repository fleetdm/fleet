import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';
import iconClassForLabel from 'utilities/icon_class_for_label';
import PlatformIcon from 'components/icons/PlatformIcon';
import statusLabelsInterface from 'interfaces/status_labels';

const baseClass = 'panel-group-item';

class PanelGroupItem extends Component {
  static propTypes = {
    item: PropTypes.shape({
      count: PropTypes.number,
      title_description: PropTypes.string,
      display_text: PropTypes.string,
      type: PropTypes.string,
    }).isRequired,
    onLabelClick: PropTypes.func,
    isSelected: PropTypes.bool,
    statusLabels: statusLabelsInterface,
    type: PropTypes.string,
  };

  displayCount = () => {
    const { item, statusLabels, type } = this.props;

    if (type !== 'status') {
      return item.count;
    }

    if (statusLabels.loading_counts) {
      return '';
    }

    return statusLabels[`${item.id}_count`];
  }

  renderIcon = () => {
    const { item, type } = this.props;

    if (type === 'platform') {
      return <PlatformIcon name={item.display_text} title={item.display_text} className={`${baseClass}__icon`} />;
    }

    return <Icon name={iconClassForLabel(item)} className={`${baseClass}__icon`} />;
  }

  renderDescription = () => {
    const { item } = this.props;
    const { title_description: titleDescription, type } = item;

    if (!titleDescription || type === 'custom') {
      return false;
    }

    return <span className={`${baseClass}__description`}>{titleDescription}</span>;
  }

  render () {
    const { displayCount, renderDescription, renderIcon } = this;
    const { item, onLabelClick, isSelected } = this.props;
    const {
      display_text: displayText,
      type,
    } = item;
    const wrapperClassName = classnames(
      baseClass,
      'button',
      'button--unstyled',
      `${baseClass}__${type.toLowerCase()}`,
      `${baseClass}__${type.toLowerCase()}--${displayText.toLowerCase().replace(' ', '-')}`,
      {
        [`${baseClass}--selected`]: isSelected,
      }
    );

    return (
      <button className={wrapperClassName} onClick={onLabelClick}>
        <div className={`${baseClass}__flexy`}>
          {renderIcon()}
          <span className={`${baseClass}__name`}>
            {displayText}
            {renderDescription()}
          </span>
          <span className={`${baseClass}__count`}>{displayCount()}</span>
        </div>
      </button>
    );
  }
}

export default PanelGroupItem;
