import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { Link } from 'react-router';
import Avatar from '../../components/Avatar';
import componentStyles from './styles';
import paths from '../../router/paths';

export class HomePage extends Component {
  static propTypes = {
    user: PropTypes.object,
  };

  render () {
    const { avatarStyles, containerStyles } = componentStyles;
    const { user } = this.props;
    const { LOGOUT } = paths;

    return (
      <div style={containerStyles}>
        <Avatar size="small" style={avatarStyles} user={user} />
        <span>You are successfully logged in! </span>
        {user && <Link to={LOGOUT}>Logout</Link>}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(HomePage);
