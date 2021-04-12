import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import statusLabelsInterface from "interfaces/status_labels";

const baseClass = "panel-group-item";

class PanelGroupItem extends Component {
  static propTypes = {
    item: PropTypes.shape({
      count: PropTypes.number.isRequired,
      title_description: PropTypes.string,
      display_text: PropTypes.string.isRequired,
      type: PropTypes.string.isRequired,
      id: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
      name: PropTypes.string,
    }).isRequired,
    onLabelClick: PropTypes.func,
    isSelected: PropTypes.bool,
    statusLabels: statusLabelsInterface,
    type: PropTypes.string,
  };

  displayCount = () => {
    const { item, statusLabels, type } = this.props;

    if (type !== "status") {
      return item.count;
    }

    if (statusLabels.loading_counts) {
      return "";
    }

    return statusLabels[`${item.id}_count`];
  };

  render() {
    const { displayCount } = this;
    const { item, onLabelClick, isSelected } = this.props;
    const { display_text: displayText, type } = item;
    const wrapperClassName = classnames(
      baseClass,
      "button",
      "button--contextual-nav-item",
      `${baseClass}__${type.toLowerCase()}`,
      `${baseClass}__${type.toLowerCase()}--${displayText
        .toLowerCase()
        .replace(" ", "-")}`,
      {
        [`${baseClass}--selected`]: isSelected,
      }
    );

    return (
      <button className={wrapperClassName} onClick={onLabelClick}>
        <div className={`${baseClass}__flexy`}>
          <span className={`${baseClass}__name`}>{displayText}</span>
          <span className={`${baseClass}__count`}>{displayCount()}</span>
        </div>
      </button>
    );
  }
}

export default PanelGroupItem;
