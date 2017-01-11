import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';
import iconClassForLabel from 'utilities/icon_class_for_label';
import PlatformIcon from 'components/icons/PlatformIcon';

const baseClass = 'panel-group-item';

class PanelGroupItem extends Component {
  static propTypes = {
    item: PropTypes.shape({
      count: PropTypes.number,
      description: PropTypes.string,
      display_text: PropTypes.string,
      type: PropTypes.string,
    }).isRequired,
    onLabelClick: PropTypes.func,
    isSelected: PropTypes.bool,
    type: PropTypes.string,
  };

  renderIcon = () => {
    const { item, type } = this.props;

    if (type === 'platform') {
      return <PlatformIcon name={item.display_text} className={`${baseClass}__icon`} />;
    }

    return <Icon name={iconClassForLabel(item)} className={`${baseClass}__icon`} />;
  }

  renderDescription = () => {
    const { item } = this.props;
    const { description, type } = item;

    if (!description || type === 'custom') {
      return false;
    }

    return <span className={`${baseClass}__description`}>{description}</span>;
  }

  render () {
    const { renderDescription, renderIcon } = this;
    const { item, onLabelClick, isSelected } = this.props;
    const {
      count,
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
          <span className={`${baseClass}__count`}>{count}</span>
        </div>
      </button>
    );
  }
}

export default PanelGroupItem;
