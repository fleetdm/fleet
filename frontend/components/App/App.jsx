import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import classnames from 'classnames';

import { authToken } from 'utilities/local';
import { fetchCurrentUser } from 'redux/nodes/auth/actions';
import FlashMessage from 'components/FlashMessage';
import { getConfig } from 'redux/nodes/app/actions';
import { hideFlash } from 'redux/nodes/notifications/actions';
import notificationInterface from 'interfaces/notification';
import userInterface from 'interfaces/user';

export class App extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    fullWidthFlash: PropTypes.bool,
    notifications: notificationInterface,
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
        .catch(() => {
          return false;
        });
    }

    if (user) {
      dispatch(getConfig());
    }

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { dispatch, user } = nextProps;

    if (user && this.props.user !== user) {
      dispatch(getConfig());
    }
  }

  onRemoveFlash = () => {
    const { dispatch } = this.props;

    dispatch(hideFlash);

    return false;
  }

  onUndoActionClick = (undoAction) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { onRemoveFlash } = this;

      dispatch(undoAction);

      return onRemoveFlash();
    };
  }

  render () {
    const { children, fullWidthFlash, notifications, showBackgroundImage } = this.props;
    const { onRemoveFlash, onUndoActionClick } = this;

    const wrapperStyles = classnames(
      'wrapper',
      { 'wrapper--background': showBackgroundImage }
    );

    return (
      <div className={wrapperStyles}>
        <FlashMessage
          fullWidth={fullWidthFlash}
          notification={notifications}
          onRemoveFlash={onRemoveFlash}
          onUndoActionClick={onUndoActionClick}
        />
        {children}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { app, auth, notifications } = state;
  const { showBackgroundImage } = app;
  const { user } = auth;
  const fullWidthFlash = !user;

  return {
    fullWidthFlash,
    notifications,
    showBackgroundImage,
    user,
  };
};

export default connect(mapStateToProps)(App);
