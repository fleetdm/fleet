import { Component, PropTypes } from 'react';

class DecoratorsPageWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
  };

  render() {
    const { children } = this.props;

    if (!children) {
      return false;
    }

    return children;
  }
}

export default DecoratorsPageWrapper;
