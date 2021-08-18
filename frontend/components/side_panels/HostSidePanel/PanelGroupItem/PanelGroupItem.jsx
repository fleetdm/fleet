import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import statusLabelsInterface from "interfaces/status_labels";
import darwinIcon from "../../../../../assets/images/icon-darwin-fleet-black-16x16@2x.png";
import linuxIcon from "../../../../../assets/images/icon-linux-fleet-black-16x16@2x.png";
import ubuntuIcon from "../../../../../assets/images/icon-ubuntu-fleet-black-16x16@2x.png";
import centosIcon from "../../../../../assets/images/icon-centos-fleet-black-16x16@2x.png";
import windowsIcon from "../../../../../assets/images/icon-windows-fleet-black-16x16@2x.png";

const baseClass = "panel-group-item";

const displayIcon = (name) => {
  switch (name) {
    case "Darwin":
      return <img src={darwinIcon} alt="Apple icon" />;
    case "Linux":
      return <img src={linuxIcon} alt="Linux icon" />;
    case "Ubuntu Linux":
      return <img src={ubuntuIcon} alt="Ubuntu icon" />;
    case "CentOS Linux":
      return <img src={centosIcon} alt="Centos icon" />;
    case "Windows":
      return <img src={windowsIcon} alt="Windows icon" />;
    default:
      break;
  }
};
class PanelGroupItem extends Component {
  static propTypes = {
    item: PropTypes.shape({
      count: PropTypes.number.isRequired,
      title_description: PropTypes.string,
      display_text: PropTypes.string.isRequired,
      type: PropTypes.string.isRequired,
      id: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
      name: PropTypes.string,
      label_type: PropTypes.string,
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
    const { display_text: displayText, type, label_type, name } = item;

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
          <span className={`${baseClass}__name`}>
            {label_type === "builtin" && displayIcon(name)}&nbsp;
            {displayText}
          </span>
          <span className={`${baseClass}__count`}>{displayCount()}</span>
        </div>
      </button>
    );
  }
}

export default PanelGroupItem;
