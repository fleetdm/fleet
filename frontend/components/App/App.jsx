import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { Style } from 'radium';
import { fetchCurrentUser } from '../../redux/nodes/auth/actions';
import Footer from './Footer';
import globalStyles from '../../styles/global';
import { authToken } from '../../utilities/local';

export class App extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    user: PropTypes.object,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount () {
    const { dispatch, user } = this.props;

    if (!user && !!authToken()) {
      dispatch(fetchCurrentUser());
    }

    return false;
  }

  render () {
    const { children } = this.props;

    return (
      <div>
        <Style rules={globalStyles} />
        {children}
        <Footer />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(App);
