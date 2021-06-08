import React, { Component } from "react";
import PropTypes from "prop-types";
import { Link } from "react-router";
import classnames from "classnames";

import FleetIcon from "components/icons/FleetIcon";

const baseClass = "stacked-white-boxes";

class StackedWhiteBoxes extends Component {
  static propTypes = {
    children: PropTypes.element,
    headerText: PropTypes.string,
    className: PropTypes.string,
    leadText: PropTypes.string,
    onLeave: PropTypes.func,
    previousLocation: PropTypes.string,
  };

  constructor(props) {
    super(props);

    this.state = {
      isLoading: false,
      isLoaded: false,
      isLeaving: false,
    };
  }

  componentWillMount() {
    this.setState({
      isLoading: true,
    });
  }

  componentDidMount() {
    const { didLoad } = this;
    didLoad();

    return false;
  }

  didLoad = () => {
    this.setState({
      isLoading: false,
      isLoaded: true,
    });
  };

  nowLeaving = (evt) => {
    const { window } = global;
    const { onLeave, previousLocation } = this.props;
    evt.preventDefault();

    this.setState({
      isLoading: false,
      isLoaded: false,
      isLeaving: true,
    });

    if (previousLocation) {
      window.setTimeout(() => {
        onLeave(previousLocation);
      }, 300);
    }

    return false;
  };

  renderBackButton = () => {
    const { previousLocation } = this.props;
    const { nowLeaving } = this;

    if (!previousLocation) return false;

    return (
      <div className={`${baseClass}__back`}>
        <Link
          to={previousLocation}
          className={`${baseClass}__back-link`}
          onClick={nowLeaving}
        >
          <FleetIcon name="x" />
        </Link>
      </div>
    );
  };

  renderHeader = () => {
    const { headerText } = this.props;

    return (
      <div className={`${baseClass}__header`}>
        <p className={`${baseClass}__header-text`}>{headerText}</p>
      </div>
    );
  };

  render() {
    const { children, className, leadText } = this.props;
    const { isLoading, isLoaded, isLeaving } = this.state;
    const { renderBackButton, renderHeader } = this;

    const boxClass = classnames(baseClass, className, {
      [`${baseClass}--loading`]: isLoading,
      [`${baseClass}--loaded`]: isLoaded,
      [`${baseClass}--leaving`]: isLeaving,
    });

    return (
      <div className={boxClass}>
        <div className={`${baseClass}__box`}>
          {renderBackButton()}
          {renderHeader()}
          <p className={`${baseClass}__box-text`}>{leadText}</p>
          {children}
        </div>
      </div>
    );
  }
}

export default StackedWhiteBoxes;
