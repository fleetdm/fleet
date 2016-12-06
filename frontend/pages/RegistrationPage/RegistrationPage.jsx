import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { max, noop } from 'lodash';
import { push } from 'react-router-redux';

import Breadcrumbs from 'pages/RegistrationPage/Breadcrumbs';
import paths from 'router/paths';
import RegistrationForm from 'components/forms/RegistrationForm';
import { setup } from 'redux/nodes/auth/actions';
import { showBackgroundImage } from 'redux/nodes/app/actions';
import userInterface from 'interfaces/user';
import Footer from 'components/Footer';

import kolideLogo from '../../../assets/images/kolide-logo-condensed.svg';

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

    this.state = {
      page: 1,
      pageProgress: 1,
    };

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
    const { page, pageProgress } = this.state;
    const nextPage = page + 1;
    this.setState({
      page: nextPage,
      pageProgress: max([nextPage, pageProgress]),
    });

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
    const { pageProgress } = this.state;
    if (page > pageProgress) {
      return false;
    }

    this.setState({ page });

    return false;
  }

  render () {
    const { isLoadingUser } = this.props;
    const { page, pageProgress } = this.state;
    const { onRegistrationFormSubmit, onNextPage, onSetPage } = this;

    if (isLoadingUser) {
      return false;
    }

    return (
      <div className="registration-page">
        <img
          alt="Kolide"
          src={kolideLogo}
          className="registration-page__logo"
        />
        <Breadcrumbs onClick={onSetPage} page={page} pageProgress={pageProgress} />
        <RegistrationForm page={page} onNextPage={onNextPage} onSubmit={onRegistrationFormSubmit} />
        <Footer />
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { loading: isLoadingUser, user: currentUser } = state.auth;

  return { currentUser, isLoadingUser };
};

export default connect(mapStateToProps)(RegistrationPage);
