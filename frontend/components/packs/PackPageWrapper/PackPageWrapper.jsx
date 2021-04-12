import React from "react";
import PropTypes from "prop-types";

class PackPageWrapper extends React.Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render() {
    const { children } = this.props;

    return children || null;
  }
}

export default PackPageWrapper;
