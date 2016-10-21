import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { StyleRoot } from 'radium';

import componentStyles from './styles';
import configInterface from '../../interfaces/config';
import FlashMessage from '../../components/FlashMessage';
import SiteNavSidePanel from '../../components/side_panels/SiteNavSidePanel';
import notificationInterface from '../../interfaces/notification';
import userInterface from '../../interfaces/user';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    notifications: notificationInterface,
    showRightSidePanel: PropTypes.bool,
    user: userInterface,
  };

  render () {
    const { children, config, dispatch, notifications, showRightSidePanel, user } = this.props;
    const { wrapperStyles } = componentStyles;

    if (!user) return false;

    const { pathname } = global.window.location;

    return (
      <StyleRoot>
        <SiteNavSidePanel
          config={config}
          pathname={pathname}
          user={user}
        />
        <div style={wrapperStyles(showRightSidePanel)}>
          <FlashMessage notification={notifications} dispatch={dispatch} />
          {children}
        </div>
      </StyleRoot>
    );
  }
}

const mapStateToProps = (state) => {
  const {
    app: { config, showRightSidePanel },
    auth: { user },
    notifications,
  } = state;

  return {
    config,
    notifications,
    showRightSidePanel,
    user,
  };
};

export default connect(mapStateToProps)(CoreLayout);
