import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import classnames from 'classnames';

import { authToken } from 'utilities/local';
import { fetchCurrentUser } from 'redux/nodes/auth/actions';
import { getConfig } from 'redux/nodes/app/actions';
import userInterface from 'interfaces/user';

export class App extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    showBackgroundImage: PropTypes.bool,
    user: userInterface,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount () {
    const { dispatch, user } = this.props;

    if (!user && !!authToken()) {
      dispatch(fetchCurrentUser())
        .catch(() => false);
    }

    if (user) {
      dispatch(getConfig())
        .catch(() => false);
    }

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { dispatch, user } = nextProps;

    if (user && this.props.user !== user) {
      dispatch(getConfig())
        .catch(() => false);
    }
  }

  render () {
    const { children, showBackgroundImage } = this.props;

    const wrapperStyles = classnames(
      'wrapper',
      { 'wrapper--background': showBackgroundImage }
    );

    return (
      <div className={wrapperStyles}>
        {children}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { app, auth } = state;
  const { showBackgroundImage } = app;
  const { user } = auth;

  return {
    showBackgroundImage,
    user,
  };
};

export default connect(mapStateToProps)(App);
