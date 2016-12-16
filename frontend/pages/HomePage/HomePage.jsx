import React, { Component } from 'react';
import { connect } from 'react-redux';
import { Link } from 'react-router';

import Avatar from 'components/Avatar';
import paths from 'router/paths';
import ProgressBar from 'components/ProgressBar';
import userInterface from 'interfaces/user';

export class HomePage extends Component {
  static propTypes = {
    user: userInterface,
  };

  render () {
    const { user } = this.props;
    const { LOGOUT } = paths;
    const baseClass = 'home-page';

    return (
      <div className={`${baseClass} body-wrap`}>
        {user && <Avatar size="small" className={`${baseClass}__avatar`} user={user} />}
        <span>You are successfully logged in! </span>
        {user && <Link to={LOGOUT}>Logout</Link>}
        <ProgressBar className={`${baseClass}__progress-bar`} max={100} value={35} />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(HomePage);
