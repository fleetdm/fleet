import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { StyleRoot } from 'radium';
import componentStyles from './styles';
import SidePanel from '../../components/SidePanel';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    user: PropTypes.object,
  };

  render () {
    const { children, user } = this.props;
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
          {children}
        </div>
      </StyleRoot>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(CoreLayout);

