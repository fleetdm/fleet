import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { push } from 'react-router-redux';

import paths from '../../router/paths';
import { setRedirectLocation } from '../../redux/nodes/redirectLocation/actions';

export class AuthenticatedRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    loading: PropTypes.bool.isRequired,
    locationBeforeTransitions: PropTypes.object,
    user: PropTypes.object,
  };

  componentWillMount () {
    const { loading, user } = this.props;
    const { redirectToLogin, redirectToPasswordReset } = this;

    if (!loading && !user) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    return false;
  }

  componentWillReceiveProps (nextProps) {
    if (isEqual(this.props, nextProps)) return false;

    const { loading, user } = nextProps;
    const { redirectToLogin, redirectToPasswordReset } = this;

    if (!loading && !user) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    return false;
  }

  redirectToLogin = () => {
    const { dispatch, locationBeforeTransitions } = this.props;
    const { LOGIN } = paths;

    dispatch(setRedirectLocation(locationBeforeTransitions));
    return dispatch(push(LOGIN));
  }

  redirectToPasswordReset = () => {
    const { dispatch } = this.props;
    const { RESET_PASSWORD } = paths;

    return dispatch(push(RESET_PASSWORD));
  }

  render () {
    const { children, user } = this.props;

    if (!user) {
      return false;
    }

    return (
      <div>
        {children}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { loading, user } = state.auth;
  const { locationBeforeTransitions } = state.routing;

  return { loading, locationBeforeTransitions, user };
};

export default connect(mapStateToProps)(AuthenticatedRoutes);
