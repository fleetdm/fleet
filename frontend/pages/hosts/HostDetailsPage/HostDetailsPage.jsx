import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import classnames from "classnames";

import { Link } from "react-router";
import ReactTooltip from "react-tooltip";
import { isEmpty, noop, pick, reduce } from "lodash";
import simpleSearch from "utilities/simple_search";
import Spinner from "components/loaders/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import SoftwareVulnerabilities from "pages/hosts/HostDetailsPage/SoftwareVulnerabilities";
import HostUsersListRow from "pages/hosts/HostDetailsPage/HostUsersListRow";
import TableContainer from "components/TableContainer";

import permissionUtils from "utilities/permissions";
import entityGetter, { memoizedGetEntity } from "redux/utilities/entityGetter";
import queryActions from "redux/nodes/entities/queries/actions";
import teamInterface from "interfaces/team";
import queryInterface from "interfaces/query";
import { renderFlash } from "redux/nodes/notifications/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import hostActions from "redux/nodes/entities/hosts/actions";
import { push } from "react-router-redux";
import PATHS from "router/paths";
import {
  Accordion,
  AccordionItem,
  AccordionItemHeading,
  AccordionItemButton,
  AccordionItemPanel,
} from "react-accessible-accordion";

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
import TransferHostModal from "./TransferHostModal";
import {
  generateTableHeaders,
  generateDataSet,
} from "./SoftwareTable/SoftwareTableConfig";
import {
  generatePackTableHeaders,
  generatePackDataSet,
} from "./PackTable/PackTableConfig";
import EmptySoftware from "./EmptySoftware";

import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";
import DeleteIcon from "../../../../assets/images/icon-action-delete-14x14@2x.png";
import TransferIcon from "../../../../assets/images/icon-action-transfer-16x16@2x.png";
import QueryIcon from "../../../../assets/images/icon-action-query-16x16@2x.png";

const baseClass = "host-details";

export class HostDetailsPage extends Component {
  static propTypes = {
    hostID: PropTypes.string,
    dispatch: PropTypes.func,
    queries: PropTypes.arrayOf(queryInterface),
    queryErrors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    isGlobalAdmin: PropTypes.bool,
    isPremiumTier: PropTypes.bool,
    isOnlyObserver: PropTypes.bool,
    canTransferTeam: PropTypes.bool,
    teams: PropTypes.arrayOf(teamInterface),
  };

  static defaultProps = {
    host: {},
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      host: {},
      isLoadingHost: true,
      showDeleteHostModal: false,
      showQueryHostModal: false,
      showTransferHostModal: false,
      showRefetchLoadingSpinner: false,
      softwareState: [],
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

    fetchHost(dispatch, hostID)
      .then((host) => {
        this.setState({
          host,
          showRefetchLoadingSpinner: host.refetch_requested,
          isLoadingHost: false,
        });
      })
      .catch((err) => {
        console.log(err);
        dispatch(
          renderFlash("error", `Unable to load host. Please try again.`)
        );
        this.setState({ isLoadingHost: false });
      });
    return false;
  }

  // Loads teams
  componentDidUpdate(prevProps) {
    const { dispatch, isPremiumTier, canTransferTeam } = this.props;
    if (
      isPremiumTier !== prevProps.isPremiumTier &&
      isPremiumTier &&
      canTransferTeam
    ) {
      dispatch(teamActions.loadAll({}));
    }
  }

  componentWillUnmount() {
    this.clearHostUpdates();
  }

  onDestroyHost = () => {
    const { dispatch } = this.props;
    const { host } = this.state;
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
    const { dispatch } = this.props;
    const { host } = this.state;
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

  onTransferHostSubmit = (team) => {
    const { toggleTransferHostModal } = this;
    const { dispatch, hostID } = this.props;
    const { fetchHost } = helpers;
    const teamId = team.id === "no-team" ? null : team.id;

    dispatch(hostActions.transferToTeam(teamId, [parseInt(hostID, 10)]))
      .then(() => {
        const successMessage =
          teamId === null
            ? `Host successfully removed from teams.`
            : `Host successfully transferred to  ${team.name}.`;
        dispatch(renderFlash("success", successMessage));
        // Update page with correct team
        this.setState({ isLoadingHost: true });
        fetchHost(dispatch, hostID)
          .then((host) => {
            this.setState({
              host,
              showRefetchLoadingSpinner: host.refetch_requested,
              isLoadingHost: false,
            });
          })
          .catch((err) => {
            console.log(err);
            dispatch(
              renderFlash("error", `Unable to load host. Please try again.`)
            );
            this.setState({ isLoadingHost: false });
          });
      })
      .catch(() => {
        dispatch(
          renderFlash("error", "Could not transfer host. Please try again.")
        );
      });
    // Must call the function and the return to avoid infinite loop
    toggleTransferHostModal()();
  };

  // Search functionality
  onQueryChange = (queryData) => {
    const { host } = this.state;
    const {
      pageIndex,
      pageSize,
      searchQuery,
      sortHeader,
      sortDirection,
    } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    if (!searchQuery) {
      this.setState({ softwareState: host.software });
      return;
    }

    this.setState({ softwareState: simpleSearch(searchQuery, host.software) });
  };

  clearHostUpdates() {
    if (this.timeout) {
      global.window.clearTimeout(this.timeout);
      this.timeout = null;
    }
  }

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

  toggleTransferHostModal = () => {
    return () => {
      const { showTransferHostModal } = this.state;

      this.setState({
        showTransferHostModal: !showTransferHostModal,
      });

      return false;
    };
  };

  renderDeleteHostModal = () => {
    const { showDeleteHostModal } = this.state;
    const { host } = this.state;
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
          <Button onClick={toggleDeleteHostModal(null)} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </Modal>
    );
  };

  renderActionButtons = () => {
    const {
      toggleDeleteHostModal,
      toggleQueryHostModal,
      toggleTransferHostModal,
    } = this;
    const { isOnlyObserver, canTransferTeam } = this.props;
    const { host } = this.state;

    const isOnline = host.status === "online";

    return (
      <div className={`${baseClass}__action-button-container`}>
        {canTransferTeam && (
          <Button
            onClick={toggleTransferHostModal()}
            variant="text-icon"
            className={`${baseClass}__transfer-button`}
          >
            Transfer <img src={TransferIcon} alt="Transfer host icon" />
          </Button>
        )}
        <div data-tip data-for="query" data-tip-disable={isOnline}>
          <Button
            onClick={toggleQueryHostModal()}
            variant="text-icon"
            disabled={!isOnline}
            className={`${baseClass}__query-button`}
          >
            Query <img src={QueryIcon} alt="Query host icon" />
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
        {!isOnlyObserver && (
          <Button onClick={toggleDeleteHostModal()} variant="text-icon">
            Delete <img src={DeleteIcon} alt="Delete host icon" />
          </Button>
        )}
      </div>
    );
  };

  renderLabels = () => {
    const { onLabelClick } = this;
    const { host } = this.state;
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
    const { host, isLoadingHost } = this.state;
    const pack_stats = host && host.pack_stats;
    const wrapperClassName = `${baseClass}__pack-table`;
    const tableHeaders = generatePackTableHeaders();

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
                <>
                  {!!pack.query_stats.length && (
                    <div className={`${wrapperClassName}`}>
                      <TableContainer
                        columns={tableHeaders}
                        data={generatePackDataSet(pack.query_stats)}
                        isLoading={isLoadingHost}
                        onQueryChange={() => null}
                        resultsTitle={"queries"}
                        defaultSortHeader={"scheduled_query_name"}
                        defaultSortDirection={"asc"}
                        showMarkAllPages={false}
                        disablePagination
                        disableCount
                      />
                    </div>
                  )}
                </>
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

  renderUsers = () => {
    const { host } = this.state;
    const { users } = host;
    const wrapperClassName = `${baseClass}__table`;

    if (users) {
      return (
        <div className="section section--users">
          <p className="section__header">Users</p>
          {users.length === 0 ? (
            <p className="results__data">
              No users were detected on this host.
            </p>
          ) : (
            <div className={`${baseClass}__wrapper`}>
              <table className={wrapperClassName}>
                <thead>
                  <tr>
                    <th>Username</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((hostUser) => {
                    return (
                      <HostUsersListRow
                        key={`host-users-row-${hostUser.username}`}
                        hostUser={hostUser}
                      />
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      );
    }
  };

  renderSoftware = () => {
    // const { EmptySoftware } = this;
    const { onQueryChange } = this;
    const { host, isLoadingHost, softwareState } = this.state;

    const tableHeaders = generateTableHeaders();

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
            <SoftwareVulnerabilities softwareList={host.software} />
            {host.software && (
              <TableContainer
                columns={tableHeaders}
                data={generateDataSet(softwareState)}
                isLoading={isLoadingHost}
                defaultSortHeader={"name"}
                defaultSortDirection={"asc"}
                inputPlaceHolder={"Filter software"}
                onQueryChange={onQueryChange}
                resultsTitle={"software items"}
                emptyComponent={EmptySoftware}
                showMarkAllPages={false}
                searchable
                wideSearch
                disablePagination
              />
            )}
          </>
        )}
      </div>
    );
  };

  renderRefetch = () => {
    const { onRefetchHost } = this;
    const { host } = this.state;
    const { showRefetchLoadingSpinner } = this.state;

    const isOnline = host.status === "online";

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
              ${!isOnline ? "refetch-offline" : ""} 
              ${showRefetchLoadingSpinner ? "refetch-spinner" : "refetch-btn"}
            `}
            disabled={!isOnline}
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
      dispatch,
      queries,
      queryErrors,
      isPremiumTier,
      isGlobalAdmin,
      teams,
      isOnlyObserver,
    } = this.props;
    const {
      host,
      isLoadingHost,
      showQueryHostModal,
      showTransferHostModal,
    } = this.state;
    const {
      toggleQueryHostModal,
      toggleTransferHostModal,
      renderDeleteHostModal,
      onTransferHostSubmit,
      renderActionButtons,
      renderLabels,
      renderSoftware,
      renderPacks,
      renderUsers,
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
        "cpu_type",
        "os_version",
        "enroll_secret_name",
        "detail_updated_at",
        "percent_disk_space_available",
        "gigs_disk_space_available",
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

    const renderDiskSpace = () => {
      if (
        host.gigs_disk_space_available > 0 ||
        host.percent_disk_space_available > 0
      ) {
        return (
          <span className="info__data">
            <div className="info__disk-space">
              <div
                className={
                  titleData.percent_disk_space_available > 20
                    ? "info__disk-space-used"
                    : "info__disk-space-warning"
                }
                style={{
                  width: `${100 - titleData.percent_disk_space_available}%`,
                }}
              />
            </div>
            {titleData.gigs_disk_space_available} GB available
          </span>
        );
      }
      return <span className="info__data">No data available</span>;
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
              {isPremiumTier && hostTeam()}
              <div className="info__item info__item--title">
                <span className="info__header">Disk Space</span>
                {renderDiskSpace()}
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">RAM</span>
                <span className="info__data">
                  {wrapKolideHelper(humanHostMemory, titleData.memory)}
                </span>
              </div>
              <div className="info__item info__item--title">
                <span className="info__header">CPU</span>
                <span className="info__data">{titleData.cpu_type}</span>
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
        {renderUsers()}
        {/* The Software inventory feature is behind a feature flag
        so we only render the sofware section if the feature is enabled */}
        {host.software && renderSoftware()}
        {renderDeleteHostModal()}
        {showQueryHostModal && (
          <SelectQueryModal
            host={host}
            onCancel={toggleQueryHostModal}
            queries={queries}
            dispatch={dispatch}
            queryErrors={queryErrors}
            isOnlyObserver={isOnlyObserver}
          />
        )}
        {showTransferHostModal && (
          <TransferHostModal
            host={host}
            onCancel={toggleTransferHostModal()}
            onSubmit={onTransferHostSubmit}
            teams={teams}
            isGlobalAdmin={isGlobalAdmin}
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
  const config = state.app.config;
  const currentUser = state.auth.user;
  const isGlobalAdmin = permissionUtils.isGlobalAdmin(currentUser);
  const isPremiumTier = permissionUtils.isPremiumTier(config);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);
  const teams = memoizedGetEntity(state.entities.teams.data);
  const canTransferTeam =
    isPremiumTier &&
    (permissionUtils.isGlobalAdmin(currentUser) ||
      permissionUtils.isGlobalMaintainer(currentUser));

  return {
    hostID,
    queries,
    queryErrors,
    isGlobalAdmin,
    isPremiumTier,
    isOnlyObserver,
    teams,
    canTransferTeam,
  };
};

export default connect(mapStateToProps)(HostDetailsPage);
