import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { max, noop } from "lodash";
import { push } from "react-router-redux";

import Breadcrumbs from "pages/RegistrationPage/Breadcrumbs";
import paths from "router/paths";
import RegistrationForm from "components/forms/RegistrationForm";
import { setup } from "redux/nodes/auth/actions";
import { showBackgroundImage } from "redux/nodes/app/actions";
import EnsureUnauthenticated from "components/EnsureUnauthenticated";

import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

export class RegistrationPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      page: 1,
      pageProgress: 1,
    };
  }

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(showBackgroundImage);

    return false;
  }

  onNextPage = () => {
    const { page, pageProgress } = this.state;
    const nextPage = page + 1;
    this.setState({
      page: nextPage,
      pageProgress: max([nextPage, pageProgress]),
    });

    return false;
  };

  onRegistrationFormSubmit = (formData) => {
    const { dispatch } = this.props;
    const { MANAGE_HOSTS } = paths;

    return dispatch(setup(formData))
      .then(() => {
        return dispatch(push(MANAGE_HOSTS));
      })
      .catch(() => {
        return false;
      });
  };

  onSetPage = (page) => {
    const { pageProgress } = this.state;
    if (page > pageProgress) {
      return false;
    }

    this.setState({ page });

    return false;
  };

  render() {
    const { page, pageProgress } = this.state;
    const { onRegistrationFormSubmit, onNextPage, onSetPage } = this;

    return (
      <div className="registration-page">
        <img
          alt="Fleet logo"
          src={fleetLogoText}
          className="registration-page__logo"
        />
        <Breadcrumbs
          onClick={onSetPage}
          page={page}
          pageProgress={pageProgress}
        />
        <RegistrationForm
          page={page}
          onNextPage={onNextPage}
          onSubmit={onRegistrationFormSubmit}
        />
      </div>
    );
  }
}

const ConnectedComponent = connect()(RegistrationPage);
export default EnsureUnauthenticated(ConnectedComponent);
