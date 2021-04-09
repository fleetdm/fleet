import React, { Component } from "react";
import PropTypes from "prop-types";

import { calculateTooltipDirection } from "./helpers";
import ClickOutside from "../../ClickOutside";

const baseClass = "ellipsis-menu";

export class EllipsisMenu extends Component {
  static propTypes = {
    children: PropTypes.node,
    positionStyles: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  };

  constructor(props) {
    super(props);

    this.state = {
      showChildren: false,
    };
  }

  componentDidMount() {
    const { setTooltipDirection } = this;

    global.window.addEventListener("resize", setTooltipDirection);

    return setTooltipDirection();
  }

  componentWillUnmount() {
    const { setTooltipDirection } = this;

    global.window.removeEventListener("resize", setTooltipDirection);

    return false;
  }

  onToggleChildren = () => {
    const { showChildren } = this.state;

    this.setState({ showChildren: !showChildren });

    return false;
  };

  setDOMNode = (DOMNode) => {
    this.DOMNode = DOMNode;
  };

  setTooltipDirection = () => {
    if (this.DOMNode) {
      const tooltipDirection = calculateTooltipDirection(this.DOMNode);

      this.setState({ tooltipDirection });
    }

    return false;
  };

  renderChildren = () => {
    const { children } = this.props;
    const { showChildren, tooltipDirection } = this.state;
    const triangleDirection = tooltipDirection === "left" ? "right" : "left";

    if (!showChildren) {
      return false;
    }

    return (
      <div
        className={`container-triangle ${triangleDirection} ${baseClass}__triangle ${baseClass}__triangle--${tooltipDirection}`}
      >
        {children}
      </div>
    );
  };

  render() {
    const { onToggleChildren, renderChildren, setDOMNode } = this;
    const { positionStyles } = this.props;

    return (
      <div ref={setDOMNode} className={baseClass} style={positionStyles}>
        <button
          onClick={onToggleChildren}
          className={`${baseClass}__btn button button--unstyled`}
        >
          &bull; &bull; &bull;
        </button>
        {renderChildren()}
      </div>
    );
  }
}

export default ClickOutside(EllipsisMenu, {
  getDOMNode: (component) => {
    return component.DOMNode;
  },
  onOutsideClick: (component) => {
    return () => {
      component.setState({ showChildren: false });

      return false;
    };
  },
});
