import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";

import helpers from "components/EmailTokenRedirect/helpers";
import userInterface from "interfaces/user";

export class EmailTokenRedirect extends Component {
  static propTypes = {
    dispatch: PropTypes.func.isRequired,
    token: PropTypes.string.isRequired,
    user: userInterface,
  };

  componentWillMount() {
    const { dispatch, token, user } = this.props;

    return helpers.confirmEmailChange(dispatch, token, user);
  }

  componentWillReceiveProps(nextProps) {
    const { dispatch, token: newToken, user: newUser } = nextProps;
    const { token: oldToken, user: oldUser } = this.props;

    const missingProps = !oldToken || !oldUser;

    if (missingProps) {
      return helpers.confirmEmailChange(dispatch, newToken, newUser);
    }

    return false;
  }

  render() {
    return <div />;
  }
}

const mapStateToProps = (state, { params }) => {
  const { token } = params;
  const { user } = state.auth;

  return { token, user };
};

export default connect(mapStateToProps)(EmailTokenRedirect);
