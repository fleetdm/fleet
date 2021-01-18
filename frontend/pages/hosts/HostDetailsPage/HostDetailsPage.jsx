import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import classnames from 'classnames';

import Spinner from 'components/loaders/Spinner';
import Button from 'components/buttons/Button';
import Modal from 'components/modals/Modal';

import entityGetter from 'redux/utilities/entityGetter';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { push } from 'react-router-redux';

import PATHS from 'router/paths';

import hostInterface from 'interfaces/host';
import { isEmpty, noop, pick } from 'lodash';
import { humanMemory, humanUptime, humanLastSeen, humanEnrolled } from 'components/hosts/HostsTable/helpers';
import helpers from './helpers';

const baseClass = 'host-details';

const dummyHost = {
  labels: [
    {
      created_at: '2021-01-04T21:18:09Z',
      updated_at: '2021-01-04T21:18:09Z',
      id: 6,
      name: 'All Hosts',
      description: 'All hosts which have enrolled in Fleet',
      query: 'select 1;',
      platform: '',
      label_type: 'builtin',
      label_membership_type: 'dynamic',
      host_count: 105,
      display_text: 'All Hosts',
      count: 105,
      host_ids: null,
    },
    {
      created_at: '2021-01-04T21:18:09Z',
      updated_at: '2021-01-04T21:18:09Z',
      id: 7,
      name: 'macOS',
      description: 'All macOS hosts',
      query: "select 1 from os_version where platform = 'darwin';",
      platform: 'darwin',
      label_type: 'builtin',
      label_membership_type: 'dynamic',
      host_count: 90,
      display_text: 'macOS',
      count: 90,
      host_ids: null,
    },
    {
      created_at: '2021-01-04T21:18:09Z',
      updated_at: '2021-01-04T21:18:09Z',
      id: 8,
      name: 'Ubuntu Linux',
      description: 'All Ubuntu hosts',
      query: "select 1 from os_version where platform = 'ubuntu';",
      platform: 'ubuntu',
      label_type: 'builtin',
      label_membership_type: 'dynamic',
      host_count: 4,
      display_text: 'Ubuntu Linux',
      count: 4,
      host_ids: null,
    },
    {
      created_at: '2021-01-04T21:18:09Z',
      updated_at: '2021-01-04T21:18:09Z',
      id: 9,
      name: 'CentOS Linux',
      description: 'All CentOS hosts',
      query: "select 1 from os_version where platform = 'centos' or name like '%centos%'",
      platform: '',
      label_type: 'builtin',
      label_membership_type: 'dynamic',
      host_count: 97,
      display_text: 'CentOS Linux',
      count: 97,
      host_ids: null,
    },
    {
      created_at: '2021-01-04T21:18:09Z',
      updated_at: '2021-01-04T21:18:09Z',
      id: 10,
      name: 'MS Windows',
      description: 'All Windows hosts',
      query: "select 1 from os_version where platform = 'windows';",
      platform: 'windows',
      label_type: 'builtin',
      label_membership_type: 'dynamic',
      host_count: 0,
      display_text: 'MS Windows',
      count: 0,
      host_ids: null,
    },
    {
      created_at: '2021-01-07T18:39:33Z',
      updated_at: '2021-01-07T18:39:33Z',
      id: 11,
      name: 'docker volumes',
      description: '',
      query: 'SELECT * FROM docker_volumes',
      platform: '',
      label_type: 'regular',
      label_membership_type: 'dynamic',
      host_count: 1,
      display_text: 'docker volumes',
      count: 1,
      host_ids: null,
    },
  ],
  packs: [
    {
      created_at: '2021-01-05T21:13:04Z',
      updated_at: '2021-01-07T19:12:54Z',
      id: 1,
      name: 'Pack',
      description: 'Pack',
      platform: '',
      disabled: true,
      query_count: 1,
      total_hosts_count: 4,
      host_ids: [],
      label_ids: [
        8,
      ],
    },
  ],
};

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

  constructor (props) {
    super(props);

    this.state = {
      showDeleteHostModal: false,
    };
  }

  componentDidMount () {
    const { dispatch, host, hostID } = this.props;
    const { fetchHost } = helpers;

    if (hostID && isEmpty(host)) {
      fetchHost(dispatch, hostID);
    }

    return false;
  }

  onQueryHost = (host) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { queryHost } = helpers;

      queryHost(dispatch, host);

      return false;
    };
  }

  onDestroyHost = (evt) => {
    evt.preventDefault();

    const { dispatch, host } = this.props;
    const { destroyHost } = helpers;

    destroyHost(dispatch, host)
      .then(() => {
        dispatch(renderFlash('success', `Host "${host.hostname}" was successfully deleted`));
      });

    return false;
  }

  onPackClick = (pack) => {
    const { dispatch } = this.props;

    return dispatch(push(PATHS.PACK({ id: pack.id })));
  }

  onLabelClick = (label) => {
    const { dispatch } = this.props;

    return dispatch(push(`${PATHS.MANAGE_HOSTS}/labels/${label.id}`));
  };

  toggleDeleteHostModal = () => {
    return () => {
      const { showDeleteHostModal } = this.state;

      this.setState({
        showDeleteHostModal: !showDeleteHostModal,
      });

      return false;
    };
  }

  renderDeleteHostModal = () => {
    const { showDeleteHostModal } = this.state;
    const { host } = this.props;
    const { toggleDeleteHostModal, onDestroyHost } = this;

    if (!showDeleteHostModal) {
      return false;
    }

    return (
      <Modal
        title="Delete Host"
        onExit={toggleDeleteHostModal(null)}
        className={`${baseClass}__modal`}
      >
        <p>This action will delete the host <strong>{host.hostname}</strong> from your Fleet instance.</p>
        <p>If the host comes back online it will automatically re-enroll. To prevent the host from re-enrolling please disable or uninstall osquery on the host.</p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={onDestroyHost} variant="alert">Delete</Button>
          <Button onClick={toggleDeleteHostModal(null)} variant="inverse">Cancel</Button>
        </div>
      </Modal>
    );
  }

  renderLabels = () => {
    const { onLabelClick } = this;
    const { labels } = dummyHost;

    const labelItems = labels.map((label) => {
      return (
        <li className="list__item" key={label.id}>
          <Button
            onClick={() => onLabelClick(label)}
            variant="label"
            className="list__button"
          >
            {label.name}
          </Button>
        </li>
      );
    });

    return (
      <div className="section labels">
        <p className="section__header">Labels</p>
        <ul className="list">
          {labelItems}
        </ul>
      </div>
    );
  }

  renderPacks = () => {
    const { onPackClick } = this;
    const { packs } = dummyHost;

    const packItems = packs.map((pack) => {
      return (
        <li className="list__item" key={pack.id}>
          <Button
            onClick={() => onPackClick(pack)}
            variant="text-link"
            className="list__button"
          >
            {pack.name}
          </Button>
        </li>
      );
    });

    return (
      <div className="section section--packs">
        <p className="section__header">Packs</p>
        <ul className="list">
          {packItems}
        </ul>
      </div>
    );
  }

  render () {
    const { host, isLoadingHost } = this.props;
    const {
      renderDeleteHostModal,
      toggleDeleteHostModal,
      onQueryHost,
      renderLabels,
      renderPacks,
    } = this;

    const titleData = pick(host, ['status', 'memory', 'host_cpu', 'os_version', 'enroll_secret_name']);
    const aboutData = pick(host, ['seen_time', 'uptime', 'last_enrolled_at', 'hardware_model', 'hardware_serial', 'primary_ip']);
    const osqueryData = pick(host, ['config_tls_refresh', 'logger_tls_period', 'distributed_interval']);
    const data = [titleData, aboutData, osqueryData];
    data.forEach((object) => {
      Object.keys(object).forEach((key) => {
        if (object[key] === '') {
          object[key] = '--';
        }
      });
    });

    const statusClassName = classnames(
      'status',
      `status--${host.status}`,
    );

    if (isLoadingHost) {
      return (
        <Spinner />
      );
    }

    return (
      <div className={`${baseClass} body-wrap`}>
        <div className="section title">
          <div className="title__inner">
            <h1 className="hostname">{host.hostname}</h1>
            <div className="info">
              <div className="info__item info__item--title">
                <span className="info__header">Status</span>
                <span className={`${statusClassName} info__data`}>{titleData.status}</span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">RAM</span>
                <span className="info__data">{humanMemory(titleData.memory)}</span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">CPU</span>
                <span className="info__data">{titleData.host_cpu}</span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">OS</span>
                <span className="info__data">{titleData.os_version}</span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">Enroll secret</span>
                <span className="info__data">{titleData.enroll_secret_name}</span>
              </div>
            </div>
          </div>
          <div>
            <Button onClick={onQueryHost(host)} variant="inverse">Query</Button>
            <Button onClick={toggleDeleteHostModal()} variant="inverse">Delete</Button>
          </div>
        </div>
        <div className="section about">
          <p className="section__header">About this host</p>
          <div className="info">
            <div className="info__item info__item--about">
              <div className="info__block">
                <span className="info__header">Last seen</span>
                <span className="info__header">Enrolled</span>
                <span className="info__header">Uptime</span>
              </div>
              <div className="info__block">
                <span className="info__data">{humanLastSeen(aboutData.seen_time)}</span>
                <span className="info__data">{humanEnrolled(aboutData.last_enrolled_at)}</span>
                <span className="info__data">{humanUptime(aboutData.uptime)}</span>
              </div>
            </div>
            <div className="info__item info__item--about">
              <div className="info__block">
                <span className="info__header">Hardware model</span>
                <span className="info__header">Serial number</span>
                <span className="info__header">IPv4</span>
              </div>
              <div className="info__block">
                <span className="info__data">{aboutData.hardware_model}</span>
                <span className="info__data">{aboutData.hardware_serial}</span>
                <span className="info__data">{aboutData.primary_ip}</span>
              </div>
            </div>
          </div>
        </div>
        <div className="section osquery">
          <p className="section__header">Osquery configuration</p>
          <div className="info">
            <div className="info__item info__item--title">
              <span className="info__header">Config TLS refresh</span>
              <span className="info__data">{osqueryData.config_tls_refresh}</span>
            </div>
            <div className="info__item info__item--title">
              <span className="info__header">Logger TLS period</span>
              <span className="info__data">{osqueryData.logger_tls_period}</span>
            </div>
            <div className="info__item info__item--title">
              <span className="info__header">Distributed interval</span>
              <span className="info__data">{osqueryData.distributed_interval}</span>
            </div>
          </div>
        </div>
        {renderLabels()}
        {renderPacks()}
        {renderDeleteHostModal()}
      </div>
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
