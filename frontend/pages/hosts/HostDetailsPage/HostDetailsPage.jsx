import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import Spinner from 'components/loaders/Spinner';

import entityGetter from 'redux/utilities/entityGetter';
import hostInterface from 'interfaces/host';
import { isEmpty, noop } from 'lodash';
import helpers from 'pages/hosts/HostDetailsPage/helpers';

export class HostDetailsPage extends Component {
  static propTypes = {
    host: hostInterface,
    hostID: PropTypes.string,
    dispatch: PropTypes.func,
    isLoadingHost: PropTypes.bool,
  }

  static defaultProps = {
    host: {},
    dispatch: noop,
  };

  componentDidMount () {
    const { dispatch, host, hostID } = this.props;
    const { fetchHost } = helpers;

    if (hostID && isEmpty(host)) {
      fetchHost(dispatch, hostID);
    }

    return false;
  }

  render () {
    const { host, isLoadingHost } = this.props;

    if (isLoadingHost) {
      return (
        <Spinner />
      );
    }

    return (
      <div>{host.hostname}</div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const { host_id: hostID } = ownProps.params;
  const host = entityGetter(state).get('hosts').findBy({ id: hostID });
  const { loading: isLoadingHost } = state.entities.hosts;
  return {
    host,
    hostID,
    isLoadingHost,
  };
};

export default connect(mapStateToProps)(HostDetailsPage);
