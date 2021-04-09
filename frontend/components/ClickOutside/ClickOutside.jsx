import React, { Component } from "react";
import { noop } from "lodash";

import { handleClickOutside } from "./helpers";

export default (
  WrappedComponent,
  { onOutsideClick = noop, getDOMNode = noop }
) => {
  class ClickOutside extends Component {
    componentDidMount() {
      const { componentInstance } = this;
      const clickHandler = onOutsideClick(componentInstance);
      const componentNode = getDOMNode(componentInstance);

      this.handleAction = handleClickOutside(clickHandler, componentNode);

      global.document.addEventListener("mousedown", this.handleAction);
      global.document.addEventListener("touchStart", this.handleAction);
    }

    componentWillUnmount() {
      global.document.removeEventListener("mousedown", this.handleAction);
      global.document.removeEventListener("touchStart", this.handleAction);
    }

    setInstance = (instance) => {
      this.componentInstance = instance;
    };

    render() {
      const { setInstance } = this;
      return <WrappedComponent {...this.props} ref={setInstance} />;
    }
  }

  return ClickOutside;
};
