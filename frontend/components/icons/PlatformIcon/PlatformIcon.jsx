import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import FleetIcon from "components/icons/FleetIcon";
import platformIconClass from "utilities/platform_icon_class";

const baseClass = "platform-icon";

export class PlatformIcon extends Component {
  static propTypes = {
    className: PropTypes.string,
    fw: PropTypes.bool,
    name: PropTypes.string.isRequired,
    size: PropTypes.string,
    title: PropTypes.string,
  };

  render() {
    const { className, name, fw, size, title } = this.props;
    const iconClasses = classnames(baseClass, className);
    let iconName = platformIconClass(name);

    if (!iconName) {
      iconName = "single-host";
    }

    return (
      <FleetIcon
        className={iconClasses}
        fw={fw}
        name={iconName}
        size={size}
        title={title}
      />
    );
  }
}

export default PlatformIcon;
