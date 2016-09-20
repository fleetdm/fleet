import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { Link } from 'react-router';
import paths from '../../router/paths';

export class HomePage extends Component {
  static propTypes = {
    user: PropTypes.object,
  };

  render () {
    const { user } = this.props;
    const { LOGOUT } = paths;

    return (
      <div>
        <i className="kolidecon-username" />
        Home page
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
