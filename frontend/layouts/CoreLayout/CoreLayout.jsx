import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import classnames from 'classnames';

import configInterface from 'interfaces/config';
import FlashMessage from 'components/FlashMessage';
import SiteNavHeader from 'components/side_panels/SiteNavHeader';
import SiteNavSidePanel from 'components/side_panels/SiteNavSidePanel';
import notificationInterface from 'interfaces/notification';
import userInterface from 'interfaces/user';

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
    const wrapperClass = classnames(
      'core-wrapper',
      { 'core-wrapper--show-panel': showRightSidePanel }
    );

    if (!user) return false;

    const { pathname } = global.window.location;

    return (
      <div>
        <nav className="site-nav">
          <SiteNavHeader
            config={config}
            user={user}
          />
          <SiteNavSidePanel
            config={config}
            pathname={pathname}
            user={user}
          />
        </nav>
        <div className={wrapperClass}>
          <FlashMessage notification={notifications} dispatch={dispatch} />
          {children}
        </div>
      </div>
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
