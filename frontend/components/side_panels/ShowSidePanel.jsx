import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop } from "lodash";

import {
  removeRightSidePanel,
  showRightSidePanel,
} from "redux/nodes/app/actions";

const ShowSidePanel = (WrappedComponent) => {
  class PageWithSidePanel extends Component {
    static propTypes = {
      dispatch: PropTypes.func,
    };

    static defaultProps = {
      dispatch: noop,
    };

    componentWillMount() {
      this.props.dispatch(showRightSidePanel);
    }

    componentWillUnmount() {
      this.props.dispatch(removeRightSidePanel);
    }

    render() {
      return <WrappedComponent {...this.props} />;
    }
  }

  return connect()(PageWithSidePanel);
};

export default ShowSidePanel;
