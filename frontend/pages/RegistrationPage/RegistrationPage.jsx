import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { push } from 'react-router-redux';

import Breadcrumbs from 'pages/RegistrationPage/Breadcrumbs';
import paths from 'router/paths';
import RegistrationForm from 'components/forms/RegistrationForm';
import { setup } from 'redux/nodes/auth/actions';
import { showBackgroundImage } from 'redux/nodes/app/actions';
import userInterface from 'interfaces/user';

export class RegistrationPage extends Component {
  static propTypes = {
    currentUser: userInterface,
    dispatch: PropTypes.func.isRequired,
    isLoadingUser: PropTypes.bool,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = { page: 1 };

    return false;
  }

  componentWillMount () {
    const { currentUser, dispatch } = this.props;
    const { HOME } = paths;

    if (currentUser) {
      dispatch(push(HOME));

      return false;
    }

    dispatch(showBackgroundImage);

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { currentUser, dispatch } = nextProps;
    const { HOME } = paths;

    if (currentUser) {
      dispatch(push(HOME));
    }
  }

  onNextPage = () => {
    const { page } = this.state;
    this.setState({ page: page + 1 });

    return false;
  }

  onRegistrationFormSubmit = (formData) => {
    const { dispatch } = this.props;
    const { LOGIN } = paths;

    return dispatch(setup(formData))
      .then(() => { return dispatch(push(LOGIN)); })
      .catch(() => { return false; });
  }

  onSetPage = (page) => {
    this.setState({ page });

    return false;
  }

  render () {
    const { isLoadingUser } = this.props;
    const { page } = this.state;
    const { onRegistrationFormSubmit, onNextPage, onSetPage } = this;

    if (isLoadingUser) {
      return false;
    }

    return (
      <div>
        <Breadcrumbs onClick={onSetPage} page={page} />
        <RegistrationForm page={page} onNextPage={onNextPage} onSubmit={onRegistrationFormSubmit} />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { loading: isLoadingUser, user: currentUser } = state.auth;

  return { currentUser, isLoadingUser };
};

export default connect(mapStateToProps)(RegistrationPage);
