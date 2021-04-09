import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop } from "lodash";

import AuthenticationFormWrapper from "../../components/AuthenticationFormWrapper";
import debounce from "../../utilities/debounce";
import LogoutForm from "../../components/forms/LogoutForm";
import { logoutUser } from "../../redux/nodes/auth/actions";
import {
  hideBackgroundImage,
  showBackgroundImage,
} from "../../redux/nodes/app/actions";
import userInterface from "../../interfaces/user";

export class LogoutPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    user: userInterface,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(showBackgroundImage);
  }

  componentWillUnmount() {
    const { dispatch } = this.props;

    dispatch(hideBackgroundImage);
  }

  onSubmit = debounce(() => {
    const { dispatch } = this.props;

    return dispatch(logoutUser());
  });

  render() {
    const { user } = this.props;
    const { onSubmit } = this;
    const baseClass = "logout-page";

    if (!user) return false;

    return (
      <div className={baseClass}>
        <AuthenticationFormWrapper>
          <div className={`${baseClass}__tab`} />
          <LogoutForm onSubmit={onSubmit} user={user} />
        </AuthenticationFormWrapper>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(LogoutPage);
