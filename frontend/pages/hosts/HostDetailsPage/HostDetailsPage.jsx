import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import Spinner from 'components/loaders/Spinner';
import Button from 'components/buttons/Button';
import Modal from 'components/modals/Modal';

import entityGetter from 'redux/utilities/entityGetter';
import hostInterface from 'interfaces/host';
import { isEmpty, noop, pick } from 'lodash';
import helpers from 'pages/hosts/HostDetailsPage/helpers';
import { renderFlash } from 'redux/nodes/notifications/actions';

const baseClass = 'host-details';

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

  render () {
    const { host, isLoadingHost } = this.props;
    const { renderDeleteHostModal, toggleDeleteHostModal, onQueryHost } = this;

    const titleData = pick(host, ['status', 'memory', 'host_cpu', 'os_version', 'enroll_secret_name']);
    const aboutData = pick(host, ['seen_time', 'last_enrolled_at', 'hardware_model', 'hardware_serial', 'primary_ip']);
    const osqueryData = pick(host, ['config_tls_refresh', 'logger_tls_period', 'distributed_interval']);

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
                <span className="info__data">{titleData.status}</span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">RAM</span>
                <span className="info__data">{titleData.memory}</span>
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
                <span className="info__data">{aboutData.seen_time}</span>
                <span className="info__data">{aboutData.last_enrolled_at}</span>
                <span className="info__data">5 hours</span>
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
          <div className="section labels">
            <p className="section__header">Labels</p>
            <ul className="list">
              <li className="list__item">
                <Button className="list__button">Label</Button>
              </li>
              <li className="list__item">
                <Button className="list__button">Label</Button>
              </li>
              <li className="list__item">
                <Button className="list__button">Label</Button>
              </li>
            </ul>
          </div>
          <div className="section section--packs">
            <p className="section__header">Packs</p>
            <ul className="list">
              <li className="list__item">
                <Button className="list__button" variant="text-link">Pack</Button>
              </li>
              <li className="list__item">
                <Button className="list__button" variant="text-link">Pack</Button>
              </li>
              <li className="list__item">
                <Button className="list__button" variant="text-link">Pack</Button>
              </li>
            </ul>
          </div>
        </div>
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
