import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { StyleRoot } from 'radium';
import componentStyles from './styles';
import FlashMessage from '../../components/FlashMessage';
import SidePanel from '../../components/SidePanel';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: PropTypes.shape({
      org_logo_url: PropTypes.string,
      org_name: PropTypes.string,
    }),
    dispatch: PropTypes.func,
    notifications: PropTypes.object,
    user: PropTypes.object,
  };

  render () {
    const { children, config, dispatch, notifications, user } = this.props;
    const { wrapperStyles } = componentStyles;

    if (!user) return false;

    const { pathname } = global.window.location;

    return (
      <StyleRoot>
        <SidePanel
          config={config}
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
  const { config } = state.app;

  return { config, notifications, user };
};

export default connect(mapStateToProps)(CoreLayout);
