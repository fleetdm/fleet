import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop } from "lodash";
import { push } from "react-router-redux";

import paths from "router/paths";
import userInterface from "interfaces/user";

export default (WrappedComponent) => {
  class EnsureUnauthenticated extends Component {
    static propTypes = {
      currentUser: userInterface,
      dispatch: PropTypes.func.isRequired,
      isLoadingUser: PropTypes.bool,
    };

    static defaultProps = {
      dispatch: noop,
    };

    componentWillMount() {
      const { currentUser, dispatch } = this.props;
      const { HOME } = paths;

      if (currentUser) {
        dispatch(push(HOME));
      }

      return false;
    }

    componentWillReceiveProps(nextProps) {
      const { currentUser, dispatch } = nextProps;
      const { HOME } = paths;

      if (currentUser) {
        dispatch(push(HOME));
      }

      return false;
    }

    render() {
      const { currentUser, isLoadingUser } = this.props;

      if (isLoadingUser || currentUser) {
        return false;
      }

      return <WrappedComponent {...this.props} />;
    }
  }

  const mapStateToProps = (state) => {
    const { loading: isLoadingUser, user: currentUser } = state.auth;

    return { currentUser, isLoadingUser };
  };

  return connect(mapStateToProps)(EnsureUnauthenticated);
};
