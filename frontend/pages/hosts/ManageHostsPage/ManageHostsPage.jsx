import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';

import componentStyles from './styles';
import entityGetter from '../../../redux/entityGetter';
import hostActions from '../../../redux/nodes/entities/hosts/actions';
import HostDetails from '../../../components/hosts/HostDetails';

class ManageHostsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    hosts: PropTypes.array,
  };

  componentWillMount () {
    const { dispatch, hosts } = this.props;

    if (!hosts.length) dispatch(hostActions.loadAll());

    return false;
  }

  onHostDetailActionClick = (type) => {
    return (host) => {
      return (evt) => {
        evt.preventDefault();

        console.log(type, host);
        return false;
      };
    };
  }

  renderHosts = () => {
    const { hosts } = this.props;
    const { onHostDetailActionClick } = this;

    return hosts.map(host => {
      return (
        <HostDetails
          host={host}
          key={host.hostname}
          onDisableClick={onHostDetailActionClick('disable')}
          onQueryClick={onHostDetailActionClick('query')}
        />
      );
    });
  }

  render () {
    const { containerStyles } = componentStyles;
    const { renderHosts } = this;

    return (
      <div style={containerStyles}>
        {renderHosts()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { entities: hosts } = entityGetter(state).get('hosts');

  return { hosts };
};

export default connect(mapStateToProps)(ManageHostsPage);
