import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';

export class AuthenticatedRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    user: PropTypes.object,
  };

  render () {
    const { children, user } = this.props;

    if (!user) return false;

    return (
      <div>
        {children}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(AuthenticatedRoutes);
