import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { noop, pick } from "lodash";

import Spinner from "components/loaders/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import SoftwareListRow from "pages/hosts/HostDetailsPage/SoftwareListRow";

import entityGetter from "redux/utilities/entityGetter";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import { renderFlash } from "redux/nodes/notifications/actions";
import { push } from "react-router-redux";
import PATHS from "router/paths";

import hostInterface from "interfaces/host";
import {
  humanHostUptime,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
} from "kolide/helpers";
import helpers from "./helpers";
import SelectQueryModal from "./SelectQueryModal";

const baseClass = "host-details";

export class HostDetailsPage extends Component {
  static propTypes = {
    host: hostInterface,
    hostID: PropTypes.string,
    dispatch: PropTypes.func,
    isLoadingHost: PropTypes.bool,
    queries: PropTypes.arrayOf(queryInterface),
    queryErrors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  };

  static defaultProps = {
    host: {},
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      showDeleteHostModal: false,
      showQueryHostModal: false,
    };
  }

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(queryActions.loadAll()).catch(() => false);

    return false;
  }

  componentDidMount() {
    const { dispatch, hostID } = this.props;
    const { fetchHost } = helpers;

    fetchHost(dispatch, hostID);

    return false;
  }

  onDestroyHost = () => {
    const { dispatch, host } = this.props;
    const { destroyHost } = helpers;

    destroyHost(dispatch, host).then(() => {
      dispatch(
        renderFlash(
          "success",
          `Host "${host.hostname}" was successfully deleted`
        )
      );
    });

    return false;
  };

  onPackClick = (pack) => {
    const { dispatch } = this.props;

    return dispatch(push(PATHS.PACK({ id: pack.id })));
  };

  onLabelClick = (label) => {
    const { dispatch } = this.props;

    return dispatch(push(`${PATHS.MANAGE_HOSTS}/labels/${label.id}`));
  };

  toggleQueryHostModal = () => {
    return () => {
      const { showQueryHostModal } = this.state;

      this.setState({
        showQueryHostModal: !showQueryHostModal,
      });

      return false;
    };
  };

  toggleDeleteHostModal = () => {
    return () => {
      const { showDeleteHostModal } = this.state;

      this.setState({
        showDeleteHostModal: !showDeleteHostModal,
      });

      return false;
    };
  };

  renderDeleteHostModal = () => {
    const { showDeleteHostModal } = this.state;
    const { host } = this.props;
    const { toggleDeleteHostModal, onDestroyHost } = this;

    if (!showDeleteHostModal) {
      return false;
    }

    return (
      <Modal
        title="Delete host"
        onExit={toggleDeleteHostModal(null)}
        className={`${baseClass}__modal`}
      >
        <p>
          This action will delete the host <strong>{host.hostname}</strong> from
          your Fleet instance.
        </p>
        <p>
          The host will automatically re-enroll when it checks back into Fleet.
        </p>
        <p>
          To prevent re-enrollment, you can uninstall osquery on the host or
          revoke the host&apos;s enroll secret.
        </p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={() => onDestroyHost()} variant="alert">
            Delete
          </Button>
          <Button onClick={toggleDeleteHostModal(null)} variant="inverse">
            Cancel
          </Button>
        </div>
      </Modal>
    );
  };

  renderActionButtons = () => {
    const { toggleDeleteHostModal, toggleQueryHostModal } = this;
    const { host } = this.props;

    const isOnline = host.status === "online";
    const isOffline = host.status === "offline";

    return (
      <div className={`${baseClass}__action-button-container`}>
        <div data-tip data-for="query" data-tip-disable={isOnline}>
          <Button
            onClick={toggleQueryHostModal()}
            variant="inverse"
            disabled={isOffline}
            className={`${baseClass}__query-button`}
          >
            Query
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="query"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t <br /> query an <br /> offline host.
          </span>
        </ReactTooltip>
        <Button onClick={toggleDeleteHostModal()} variant="inverse">
          Delete
        </Button>
      </div>
    );
  };

  renderLabels = () => {
    const { onLabelClick } = this;
    const { host } = this.props;
    const { labels = [] } = host;

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
        {labels.length === 0 ? (
          <p className="info__item">No labels are associated with this host.</p>
        ) : (
          <ul className="list">{labelItems}</ul>
        )}
      </div>
    );
  };

  renderPacks = () => {
    const { onPackClick } = this;
    const { host } = this.props;
    const { packs = [] } = host;

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
        {packs.length === 0 ? (
          <p className="info__item">No packs have this host as a target.</p>
        ) : (
          <ul className="list">{packItems}</ul>
        )}
      </div>
    );
  };

  renderSoftware = () => {
    const { host } = this.props;
    const wrapperClassName = `${baseClass}__table`;

    return (
      <div className="section section--software">
        <p className="section__header">Software</p>
        {host.software.length === 0 ? (
          <div className="results">
            <p className="results__header">
              No installed software detected on this host.
            </p>
            <p className="results__data">
              Expecting to see software? Try again in a few seconds as the
              system catches up.
            </p>
          </div>
        ) : (
          <div className={`${baseClass}__wrapper`}>
            <table className={wrapperClassName}>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Installed Version</th>
                </tr>
              </thead>
              <tbody>
                {!!host.software.length &&
                  host.software.map((software) => {
                    return (
                      <SoftwareListRow
                        key={`software-row-${software.id}`}
                        software={software}
                      />
                    );
                  })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    );
  };

  render() {
    const { host, isLoadingHost, dispatch, queries, queryErrors } = this.props;
    const { showQueryHostModal } = this.state;
    const {
      toggleQueryHostModal,
      renderDeleteHostModal,
      renderActionButtons,
      renderLabels,
      renderSoftware,
      renderPacks,
    } = this;

    const titleData = pick(host, [
      "status",
      "memory",
      "host_cpu",
      "os_version",
      "enroll_secret_name",
      "detail_updated_at",
    ]);
    const aboutData = pick(host, [
      "seen_time",
      "uptime",
      "last_enrolled_at",
      "hardware_model",
      "hardware_serial",
      "primary_ip",
    ]);
    const osqueryData = pick(host, [
      "config_tls_refresh",
      "logger_tls_period",
      "distributed_interval",
    ]);
    const data = [titleData, aboutData, osqueryData];
    data.forEach((object) => {
      Object.keys(object).forEach((key) => {
        if (object[key] === "") {
          object[key] = "--";
        }
      });
    });

    const statusClassName = classnames("status", `status--${host.status}`);

    if (isLoadingHost) {
      return <Spinner />;
    }

    return (
      <div className={`${baseClass} body-wrap`}>
        <div className="section title">
          <div className="title__inner">
            <div className="hostname-container">
              <h1 className="hostname">{host.hostname}</h1>
              <p className="last-fetched">{`Last fetched ${humanHostDetailUpdated(
                titleData.detail_updated_at
              )}`}</p>
            </div>
            <div className="info">
              <div className="info__item info__item--title">
                <span className="info__header">Status</span>
                <span className={`${statusClassName} info__data`}>
                  {titleData.status}
                </span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">RAM</span>
                <span className="info__data">
                  {humanHostMemory(titleData.memory)}
                </span>
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
                <span className="info__data">
                  {titleData.enroll_secret_name}
                </span>
              </div>
            </div>
          </div>
          {renderActionButtons()}
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
                <span className="info__data">
                  {humanHostLastSeen(aboutData.seen_time)}
                </span>
                <span className="info__data">
                  {humanHostEnrolled(aboutData.last_enrolled_at)}
                </span>
                <span className="info__data">
                  {humanHostUptime(aboutData.uptime)}
                </span>
              </div>
            </div>
            <div className="info__item info__item--about">
              <div className="info__block">
                <span className="info__header">Hardware model</span>
                <span className="info__header">Serial number</span>
                <span className="info__header">IP address</span>
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
              <span className="info__header">Config refresh</span>
              <span className="info__data">
                {osqueryData.config_tls_refresh}
              </span>
            </div>
            <div className="info__item info__item--title">
              <span className="info__header">Logger TLS period</span>
              <span className="info__data">
                {osqueryData.logger_tls_period}
              </span>
            </div>
            <div className="info__item info__item--title">
              <span className="info__header">Distributed interval</span>
              <span className="info__data">
                {osqueryData.distributed_interval}
              </span>
            </div>
          </div>
        </div>
        {renderLabels()}
        {renderPacks()}
        {host.software && renderSoftware()}
        {renderDeleteHostModal()}
        {showQueryHostModal && (
          <SelectQueryModal
            host={host}
            toggleQueryHostModal={toggleQueryHostModal}
            queries={queries}
            dispatch={dispatch}
            queryErrors={queryErrors}
          />
        )}
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const queryEntities = entityGetter(state).get("queries");
  const { entities: queries, errors: queryErrors } = queryEntities;
  const { host_id: hostID } = ownProps.params;
  const host = entityGetter(state).get("hosts").findBy({ id: hostID });
  const { loading: isLoadingHost } = state.entities.hosts;
  return {
    host,
    hostID,
    isLoadingHost,
    queries,
    queryErrors,
  };
};

export default connect(mapStateToProps)(HostDetailsPage);
