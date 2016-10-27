import React, { Component, PropTypes } from 'react';

class ShadowBox extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div className="shadow-box__wrapper">
        {children}
      </div>
    );
  }
}

export default ShadowBox;
