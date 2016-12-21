import React from 'react';

class PackPageWrapper extends React.Component {
  static propTypes = {
    children: React.PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (children || null);
  }
}

export default PackPageWrapper;
