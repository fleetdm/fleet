import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import classnames from "classnames";

import { Link } from "react-router";
import ReactTooltip from "react-tooltip";
import { isEmpty, noop, pick, reduce } from "lodash";

import FleetIcon from "components/icons/FleetIcon";
import Spinner from "components/loaders/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import SoftwareListRow from "pages/hosts/HostDetailsPage/SoftwareListRow";
import PackQueriesListRow from "pages/hosts/HostDetailsPage/PackQueriesListRow";

import permissionUtils from "utilities/permissions";
import entityGetter from "redux/utilities/entityGetter";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import { renderFlash } from "redux/nodes/notifications/actions";
import { push } from "react-router-redux";
import PATHS from "router/paths";
import {
  Accordion,
  AccordionItem,
  AccordionItemHeading,
  AccordionItemButton,
  AccordionItemPanel,
} from "react-accessible-accordion";

import hostInterface from "interfaces/host";
import {
  humanHostUptime,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
  secondsToHms,
} from "fleet/helpers";
import helpers from "./helpers";
import SelectQueryModal from "./SelectQueryModal";

import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";

const baseClass = "host-details";

export class HostDetailsPage extends Component {
  static propTypes = {
    host: hostInterface,
    hostID: PropTypes.string,
    dispatch: PropTypes.func,
    isLoadingHost: PropTypes.bool,
    queries: PropTypes.arrayOf(queryInterface),
    queryErrors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    isBasicTier: PropTypes.bool,
    isOnlyObserver: PropTypes.bool,
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
      showRefetchLoadingSpinner: false,
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

    fetchHost(dispatch, hostID).then((host) =>
      this.setState({ showRefetchLoadingSpinner: host.refetch_requested })
    );
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

  onRefetchHost = () => {
    const { dispatch, host } = this.props;
    const { refetchHost } = helpers;

    this.setState({ showRefetchLoadingSpinner: true });

    refetchHost(dispatch, host).catch((error) => {
      this.setState({ showRefetchLoadingSpinner: false });
      console.log(error);
      dispatch(renderFlash("error", `Host "${host.hostname}" refetch error`));
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
          <Button onClick={onDestroyHost} variant="alert">
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
    const { host, isOnlyObserver } = this.props;

    const isOnline = host.status === "online";
    const isOffline = host.status === "offline";

    // Hide action buttons for global and team only observers
    if (isOnlyObserver) {
      return null;
    }

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
            You can’t query <br /> an offline host.
          </span>
        </ReactTooltip>
        <Button onClick={toggleDeleteHostModal()} variant="active">
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
      <div className="section labels col-25">
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
    const { host } = this.props;
    const { pack_stats } = host;
    const wrapperClassName = `${baseClass}__table`;

    let packsAccordion;
    if (pack_stats) {
      packsAccordion = pack_stats.map((pack) => {
        return (
          <AccordionItem key={pack.pack_id}>
            <AccordionItemHeading>
              <AccordionItemButton>{pack.pack_name}</AccordionItemButton>
            </AccordionItemHeading>
            <AccordionItemPanel>
              {pack.query_stats.length === 0 ? (
                <div>There are no schedule queries for this pack.</div>
              ) : (
                <div className={`${baseClass}__wrapper`}>
                  <table className={wrapperClassName}>
                    <thead>
                      <tr>
                        <th>Query name</th>
                        <th>Description</th>
                        <th>Frequency</th>
                        <th>Last run</th>
                      </tr>
                    </thead>
                    <tbody>
                      {!!pack.query_stats.length &&
                        pack.query_stats.map((query) => {
                          return (
                            <PackQueriesListRow
                              key={`pack-row-${query.pack_id}-${query.scheduled_query_id}`}
                              query={query}
                            />
                          );
                        })}
                    </tbody>
                  </table>
                </div>
              )}
            </AccordionItemPanel>
          </AccordionItem>
        );
      });
    }

    return (
      <div className="section section--packs">
        <p className="section__header">Packs</p>
        {!pack_stats ? (
          <p className="results__data">
            No packs with scheduled queries have this host as a target.
          </p>
        ) : (
          <Accordion allowMultipleExpanded="true" allowZeroExpanded="true">
            {packsAccordion}
          </Accordion>
        )}
      </div>
    );
  };

  renderSoftware = () => {
    const { host } = this.props;
    const wrapperClassName = `${baseClass}__table`;

    const softwarez = [
      {
        id: 1,
        name: "Figma.app",
        version: "4.2.0",
        source: "apps",
        generated_cpe: "",
        vulnerabilities: [],
      },
      {
        id: 2,
        name: "Google Chrome.app",
        version: "91.0.4472.101",
        source: "apps",
        generated_cpe: "cpe:2.3:a:google:chrome:91.0.4472.77:*:*:*:*:*:*:*",
        vulnerabilities: [
          {
            cve: "CVE-2013-6662",
            details_link: "https://nvd.nist.gov/vuln/detail/CVE-2013-6662",
          },
          {
            cve: "CVE-2014-6662",
            details_link: "https://nvd.nist.gov/vuln/detail/CVE-2014-6662",
          },
          {
            cve: "CVE-2015-6662",
            details_link: "https://nvd.nist.gov/vuln/detail/CVE-2015-6662",
          },
        ],
      },
      {
        id: 3,
        name: "Make Believe.app",
        version: "91.0.4472.101",
        source: "apps",
        generated_cpe: "cpe:2.3:a:google:chrome:91.0.4472.77:*:*:*:*:*:*:*",
        vulnerabilities: [
          {
            cve: "CVE-2016-6662",
            details_link: "https://nvd.nist.gov/vuln/detail/CVE-2016-6662",
          },
        ],
      },
    ];

    let vulsList = [];

    const vulnerabilitiesListMaker = (softwarezz) => {
      softwarezz.forEach((software) => {
        let softwareName = software.name;
        software.vulnerabilities.forEach((vulnerability) => {
          vulsList.push({
            name: softwareName,
            cve: vulnerability.cve,
            details_link: vulnerability.details_link,
          });
        });
      });
    };

    vulnerabilitiesListMaker(softwarez);

    const renderVulsCount = (list) => {
      if (list.length === 1) {
        return "1 vulnerability detected";
      }
      if (list.length > 1) {
        return `${list.length} vulnerabilities detected`;
      }
    };

    const renderVul = (vul, index) => {
      return (
        <li key={index}>
          Read more about{" "}
          <a href={vul.details_link} target="_blank" rel="noopener noreferrer">
            <em>{vul.name}</em> {vul.cve} vulnerability{" "}
            <FleetIcon name="external-link" />
          </a>
        </li>
      );
    };

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
          <>
            <div className={`${baseClass}__vul-wrapper`}>
              <div className={`${baseClass}__vul-count`}>
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 16 16"
                  fill="none"
                >
                  <path
                    d="M0 8C0 12.4183 3.5817 16 8 16C12.4183 16 16 12.4183 16 8C16 3.5817 12.4183 0 8 0C3.5817 0 0 3.5817 0 8ZM14 8C14 11.3137 11.3137 14 8 14C4.6863 14 2 11.3137 2 8C2 4.6863 4.6863 2 8 2C11.3137 2 14 4.6863 14 8ZM7 12V10H9V12H7ZM7 4V9H9V4H7Z"
                    fill="#8B8FA2"
                  />
                </svg>{" "}
                {renderVulsCount(vulsList)}
              </div>
              <div className={`${baseClass}__vul-list`}>
                <ul>{vulsList.map((vul, index) => renderVul(vul, index))}</ul>
              </div>
            </div>
            <div className={`${baseClass}__wrapper`}>
              <table className={wrapperClassName}>
                <thead>
                  <tr>
                    <th></th>
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
          </>
        )}
      </div>
    );
  };

  renderRefetch = () => {
    const { onRefetchHost } = this;
    const { host } = this.props;
    const { showRefetchLoadingSpinner } = this.state;

    const isOnline = host.status === "online";
    const isOffline = host.status === "offline";
    return (
      <>
        <div
          className="refetch"
          data-tip
          data-for="refetch-tooltip"
          data-tip-disable={isOnline || showRefetchLoadingSpinner}
        >
          <Button
            className={`
              button
              button--unstyled
              ${isOffline ? "refetch-offline" : ""} 
              ${showRefetchLoadingSpinner ? "refetch-spinner" : "refetch-btn"}
            `}
            disabled={isOffline}
            onClick={onRefetchHost}
          >
            {showRefetchLoadingSpinner
              ? "Fetching, try refreshing this page in just a moment."
              : "Refetch"}
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="refetch-tooltip"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t fetch data from <br /> an offline host.
          </span>
        </ReactTooltip>
      </>
    );
  };

  render() {
    const {
      host,
      isLoadingHost,
      dispatch,
      queries,
      queryErrors,
      isBasicTier,
    } = this.props;
    const { showQueryHostModal } = this.state;
    const {
      toggleQueryHostModal,
      renderDeleteHostModal,
      renderActionButtons,
      renderLabels,
      renderSoftware,
      renderPacks,
      renderRefetch,
    } = this;

    const normalizeEmptyValues = (hostData) => {
      return reduce(
        hostData,
        (result, value, key) => {
          if ((Number.isFinite(value) && value !== 0) || !isEmpty(value)) {
            Object.assign(result, { [key]: value });
          } else {
            Object.assign(result, { [key]: "---" });
          }
          return result;
        },
        {}
      );
    };

    const wrapKolideHelper = (helperFn, value) => {
      return value === "---" ? value : helperFn(value);
    };

    const titleData = normalizeEmptyValues(
      pick(host, [
        "status",
        "memory",
        "host_cpu",
        "os_version",
        "enroll_secret_name",
        "detail_updated_at",
      ])
    );
    const aboutData = normalizeEmptyValues(
      pick(host, [
        "seen_time",
        "uptime",
        "last_enrolled_at",
        "hardware_model",
        "hardware_serial",
        "primary_ip",
      ])
    );
    const osqueryData = normalizeEmptyValues(
      pick(host, [
        "config_tls_refresh",
        "logger_tls_period",
        "distributed_interval",
      ])
    );

    const statusClassName = classnames("status", `status--${host.status}`);

    if (isLoadingHost) {
      return <Spinner />;
    }

    const hostTeam = () => {
      return (
        <div className="info__item info__item--title">
          <span className="info__header">Team</span>
          <span className={`info__data`}>
            {host.team_name ? (
              `${host.team_name}`
            ) : (
              <span className="info__no-team">No team</span>
            )}
          </span>
        </div>
      );
    };
    return (
      <div className={`${baseClass} body-wrap`}>
        <div>
          <Link to={PATHS.MANAGE_HOSTS} className={`${baseClass}__back-link`}>
            <img src={BackChevron} alt="back chevron" id="back-chevron" />
            <span>Back to all hosts</span>
          </Link>
        </div>
        <div className="section title">
          <div className="title__inner">
            <div className="hostname-container">
              <h1 className="hostname">
                {host.hostname ? host.hostname : "---"}
              </h1>
              <p className="last-fetched">
                {`Last fetched ${humanHostDetailUpdated(
                  titleData.detail_updated_at
                )}`}{" "}
              </p>
              {renderRefetch()}
            </div>
            <div className="info">
              <div className="info__item info__item--title">
                <span className="info__header">Status</span>
                <span className={`${statusClassName} info__data`}>
                  {titleData.status}
                </span>
              </div>
              {isBasicTier ? hostTeam() : null}
              <div className="info__item info__item--title">
                <span className="info__header">RAM</span>
                <span className="info__data">
                  {wrapKolideHelper(humanHostMemory, titleData.memory)}
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
            </div>
          </div>
          {renderActionButtons()}
        </div>
        <div className="section about col-50">
          <p className="section__header">About this host</p>
          <div className="info">
            <div className="info__item info__item--about">
              <div className="info__block">
                <span className="info__header">Created at</span>
                <span className="info__data">
                  {wrapKolideHelper(
                    humanHostEnrolled,
                    aboutData.last_enrolled_at
                  )}
                </span>
                <span className="info__header">Updated at</span>
                <span className="info__data">
                  {wrapKolideHelper(
                    humanHostLastSeen,
                    titleData.detail_updated_at
                  )}
                </span>
                <span className="info__header">Uptime</span>
                <span className="info__data">
                  {wrapKolideHelper(humanHostUptime, aboutData.uptime)}
                </span>
              </div>
            </div>
            <div className="info__item info__item--about">
              <div className="info__block">
                <span className="info__header">Hardware model</span>
                <span className="info__data">{aboutData.hardware_model}</span>
                <span className="info__header">Serial number</span>
                <span className="info__data">{aboutData.hardware_serial}</span>
                <span className="info__header">IPv4</span>
                <span className="info__data">{aboutData.primary_ip}</span>
              </div>
            </div>
          </div>
        </div>
        <div className="section osquery col-25">
          <p className="section__header">Agent options</p>
          <div className="info__item info__item--about">
            <div className="info__block">
              <span className="info__header">Config TLS refresh</span>
              <span className="info__data">
                {wrapKolideHelper(secondsToHms, osqueryData.config_tls_refresh)}
              </span>
              <span className="info__header">Logger TLS period</span>
              <span className="info__data">
                {wrapKolideHelper(secondsToHms, osqueryData.logger_tls_period)}
              </span>
              <span className="info__header">Distributed interval</span>
              <span className="info__data">
                {wrapKolideHelper(
                  secondsToHms,
                  osqueryData.distributed_interval
                )}
              </span>
            </div>
          </div>
        </div>
        {renderLabels()}
        {renderPacks()}
        {/* The Software inventory feature is behind a feature flag
        so we only render the sofware section if the feature is enabled */}
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
  const config = state.app.config;
  const currentUser = state.auth.user;
  const isBasicTier = permissionUtils.isBasicTier(config);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);

  return {
    host,
    hostID,
    isLoadingHost,
    queries,
    queryErrors,
    isBasicTier,
    isOnlyObserver,
  };
};

export default connect(mapStateToProps)(HostDetailsPage);
