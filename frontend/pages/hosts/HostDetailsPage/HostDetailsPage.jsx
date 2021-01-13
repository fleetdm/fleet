import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import entityGetter from 'redux/utilities/entityGetter';
import hostInterface from 'interfaces/host';
import { isEmpty } from 'lodash';
import helpers from 'pages/hosts/HostDetailsPage/helpers';

export class HostDetailsPage extends Component {
  static propTypes = {
    host: hostInterface,
    hostID: PropTypes.string,
    dispatch: PropTypes.func,

  }

  static defaultProps = {
    host: {},
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
    const { host } = this.props;
    return (
      <div>{host.hostname}</div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const { host_id: hostID } = ownProps.params;
  const host = entityGetter(state).get('hosts').findBy({ id: hostID });
  return {
    host,
    hostID,
  };
};

export default connect(mapStateToProps)(HostDetailsPage);
