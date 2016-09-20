import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { isEqual } from 'lodash';
import { push } from 'react-router-redux';
import paths from '../../router/paths';

export class AuthenticatedRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    loading: PropTypes.bool.isRequired,
    user: PropTypes.object,
  };

  componentWillMount () {
    const { loading, user } = this.props;
    const { redirectToLogin } = this;

    if (!loading && !user) return redirectToLogin();

    return false;
  }

  componentWillReceiveProps (nextProps) {
    if (isEqual(this.props, nextProps)) return false;

    const { loading, user } = nextProps;
    const { redirectToLogin } = this;

    if (!loading && !user) return redirectToLogin();

    return false;
  }

  redirectToLogin = () => {
    const { dispatch } = this.props;
    const { LOGIN } = paths;

    return dispatch(push(LOGIN));
  }

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
  const { loading, user } = state.auth;

  return { loading, user };
};

export default connect(mapStateToProps)(AuthenticatedRoutes);
