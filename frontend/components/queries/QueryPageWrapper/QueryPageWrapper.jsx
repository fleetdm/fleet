import { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";

import entityGetter from "redux/utilities/entityGetter";
import helpers from "components/queries/QueryPageWrapper/helpers";
import queryInterface from "interfaces/query";

class QueryPageWrapper extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    query: queryInterface,
    queryID: PropTypes.string,
  };

  componentDidMount() {
    const { dispatch, query, queryID } = this.props;
    const { fetchQuery } = helpers;

    if (queryID && !query) {
      fetchQuery(dispatch, queryID);
    }

    return false;
  }

  render() {
    const { children } = this.props;

    if (!children) {
      return false;
    }

    return children;
  }
}

const mapStateToProps = (state, { params }) => {
  const { id: queryID } = params;
  const query = entityGetter(state).get("queries").findBy({ id: queryID });

  return { query, queryID };
};

export default connect(mapStateToProps)(QueryPageWrapper);
