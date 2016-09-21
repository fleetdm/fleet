import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import SidePanel from '../../components/SidePanel';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    user: PropTypes.object,
  };

  render () {
    const { children, user } = this.props;

    if (!user) return false;

    return (
      <div>
        <SidePanel user={user} />
        <div style={{ marginLeft: '240px' }}>{children}</div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(CoreLayout);

