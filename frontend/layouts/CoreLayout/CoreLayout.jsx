import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { StyleRoot } from 'radium';
import componentStyles from './styles';
import FlashMessage from '../../components/FlashMessage';
import SidePanel from '../../components/SidePanel';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    notifications: PropTypes.object,
    user: PropTypes.object,
  };

  render () {
    const { children, dispatch, notifications, user } = this.props;
    const { wrapperStyles } = componentStyles;

    if (!user) return false;

    const { pathname } = global.window.location;

    return (
      <StyleRoot>
        <SidePanel
          pathname={pathname}
          user={user}
        />
        <div style={wrapperStyles}>
          <FlashMessage notification={notifications} dispatch={dispatch} />
          {children}
        </div>
      </StyleRoot>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;
  const { notifications } = state;

  return { user, notifications };
};

export default connect(mapStateToProps)(CoreLayout);
