import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import fleetAvatar from "../../../../assets/images/fleet-avatar-24x24@2x.png";

const baseClass = "org-logo-icon";

class OrgLogoIcon extends Component {
  static propTypes = {
    className: PropTypes.string,
    src: PropTypes.string.isRequired,
  };

  static defaultProps = {
    src: fleetAvatar,
  };

  constructor(props) {
    super(props);

    this.state = { imageSrc: props.src || fleetAvatar, prevSrc: props.src };
  }

  static getDerivedStateFromProps(nextProps, prevState) {
    if (nextProps.src !== prevState.prevSrc) {
      return {
        imageSrc: nextProps.src || fleetAvatar,
        prevSrc: nextProps.src,
      };
    }
    return null;
  }

  onError = () => {
    this.setState({ imageSrc: fleetAvatar });
  };

  render() {
    const { className } = this.props;
    const { imageSrc } = this.state;
    const { onError } = this;

    const classNames =
      imageSrc === fleetAvatar
        ? classnames(baseClass, className, "default-fleet-logo")
        : classnames(baseClass, className);

    return (
      <img
        key={imageSrc}
        alt="Organization Logo"
        className={classNames}
        onError={onError}
        src={imageSrc}
      />
    );
  }
}

export default OrgLogoIcon;
