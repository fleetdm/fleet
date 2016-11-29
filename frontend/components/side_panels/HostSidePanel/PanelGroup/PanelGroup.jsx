import React, { Component, PropTypes } from 'react';
import { isEqual, noop } from 'lodash';

import labelInterface from 'interfaces/label';
import PanelGroupItem from '../PanelGroupItem';

class PanelGroup extends Component {
  static propTypes = {
    groupItems: PropTypes.arrayOf(labelInterface),
    onLabelClick: PropTypes.func,
    selectedLabel: labelInterface,
  };

  static defaultProps = {
    onLabelClick: noop,
  };

  renderGroupItem = (item) => {
    const {
      onLabelClick,
      selectedLabel,
    } = this.props;
    const selected = isEqual(selectedLabel, item);

    return (
      <PanelGroupItem
        isSelected={selected}
        item={item}
        key={item.display_text}
        onLabelClick={onLabelClick(item)}
      />
    );
  }

  render () {
    const { groupItems } = this.props;
    const { renderGroupItem } = this;
    const baseClass = 'panel-group';

    return (
      <div className={baseClass}>
        {groupItems.map((item) => {
          return renderGroupItem(item);
        })}
      </div>
    );
  }
}

export default PanelGroup;
