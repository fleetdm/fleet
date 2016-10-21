import React, { Component, PropTypes } from 'react';

class QueryPageWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render () {
    const { children } = this.props;

    return (
      <div>
        {children}
      </div>
    );
  }
}

export default QueryPageWrapper;
