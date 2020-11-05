import React, { PureComponent } from 'react';
import PropTypes from 'prop-types';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import { sortBy } from 'lodash';
import classNames from 'classnames';

import AddHostModal from 'components/hosts/AddHostModal';
import Button from 'components/buttons/Button';
import configInterface from 'interfaces/config';
import HostContainer from 'components/hosts/HostContainer';
import HostPagination from 'components/hosts/HostPagination';
import HostSidePanel from 'components/side_panels/HostSidePanel';
import Icon from 'components/icons/Icon';
import LabelForm from 'components/forms/LabelForm';
import Modal from 'components/modals/Modal';
import PlatformIcon from 'components/icons/PlatformIcon';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import labelInterface from 'interfaces/label';
import hostInterface from 'interfaces/host';
import osqueryTableInterface from 'interfaces/osquery_table';
import statusLabelsInterface from 'interfaces/status_labels';
import enrollSecretInterface from 'interfaces/enroll_secret';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import {
  getStatusLabelCounts,
  setDisplay,
  setPagination,
} from 'redux/nodes/components/ManageHostsPage/actions';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';
import entityGetter from 'redux/utilities/entityGetter';
import PATHS from 'router/paths';
import deepDifference from 'utilities/deep_difference';
import iconClassForLabel from 'utilities/icon_class_for_label';
import platformIconClass from 'utilities/platform_icon_class';
import scrollToTop from 'utilities/scroll_to_top';
import helpers from './helpers';

const NEW_LABEL_HASH = '#new_label';
const baseClass = 'manage-hosts';

export class ManageHostsPage extends PureComponent {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func,
    display: PropTypes.oneOf(['Grid', 'List']),
    hosts: PropTypes.arrayOf(hostInterface),
    isAddLabel: PropTypes.bool,
    labelErrors: PropTypes.shape({
      base: PropTypes.string,
    }),
    labels: PropTypes.arrayOf(labelInterface),
    loadingHosts: PropTypes.bool.isRequired,
    loadingLabels: PropTypes.bool.isRequired,
    enrollSecret: enrollSecretInterface,
    selectedFilter: PropTypes.string,
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
    page: PropTypes.number,
    perPage: PropTypes.number,
  };

  static defaultProps = {
    display: 'Grid',
    page: 1,
    perPage: 100,
    loadingHosts: false,
    loadingLabels: false,
  };

  constructor (props) {
    super(props);

    this.state = {
      isEditLabel: false,
      labelQueryText: '',
      pagedHosts: [],
      showDeleteHostModal: false,
      showAddHostModal: false,
      selectedHost: null,
      showDeleteLabelModal: false,
      showHostContainerSpinner: false,
    };
  }

  componentDidMount () {
    const { dispatch, page, perPage, selectedFilter } = this.props;

    dispatch(setPagination(page, perPage, selectedFilter));
  }

  componentWillUnmount () {
    this.clearHostUpdates();
    return false;
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(push(`${PATHS.MANAGE_HOSTS}${NEW_LABEL_HASH}`));

    return false;
  }

  onCancelAddLabel = () => {
    const { dispatch } = this.props;

    dispatch(push(PATHS.MANAGE_HOSTS));

    return false;
  }

  onAddHostClick = (evt) => {
    evt.preventDefault();

    const { toggleAddHostModal } = this;
    toggleAddHostModal();

    return false;
  }

  onDestroyHost = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;
    const { selectedHost } = this.state;

    dispatch(hostActions.destroy(selectedHost))
      .then(() => {
        this.toggleDeleteHostModal(null)();

        dispatch(getStatusLabelCounts);
        dispatch(renderFlash('success', `Host "${selectedHost.hostname}" was successfully deleted`));
      });

    return false;
  }

  onEditLabel = (formData) => {
    const { dispatch, selectedLabel } = this.props;
    const updateAttrs = deepDifference(formData, selectedLabel);

    return dispatch(labelActions.update(selectedLabel, updateAttrs))
      .then(() => {
        this.toggleEditLabel();

        return false;
      })
      .catch(() => false);
  }

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch, perPage } = this.props;
      const { MANAGE_HOSTS } = PATHS;
      const { slug, type } = selectedLabel;
      const nextLocation = type === 'all' ? MANAGE_HOSTS : `${MANAGE_HOSTS}/${slug}`;

      dispatch(push(nextLocation));

      dispatch(setPagination(1, perPage, selectedLabel.slug));

      return false;
    };
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  }

  onPaginationChange = (page) => {
    const { dispatch, selectedFilter, perPage } = this.props;

    dispatch(setPagination(page, perPage, selectedFilter));

    scrollToTop();

    return true;
  }

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData))
      .then(() => {
        dispatch(push(PATHS.MANAGE_HOSTS));

        return false;
      });
  }

  onToggleDisplay = (event) => {
    event.preventDefault();
    const { dispatch } = this.props;
    const value = event.currentTarget.dataset.value;

    dispatch(setDisplay(value));

    return false;
  }

  onDeleteLabel = () => {
    const { toggleDeleteLabelModal } = this;
    const { dispatch, selectedLabel } = this.props;
    const { MANAGE_HOSTS } = PATHS;

    return dispatch(labelActions.destroy(selectedLabel))
      .then(() => {
        toggleDeleteLabelModal();
        dispatch(push(MANAGE_HOSTS));
        return false;
      });
  }

  onQueryHost = (host) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { NEW_QUERY } = PATHS;

      dispatch(push({
        pathname: NEW_QUERY,
        query: { host_ids: [host.id] },
      }));

      return false;
    };
  }

  clearHostUpdates () {
    if (this.timeout) {
      global.window.clearTimeout(this.timeout);
      this.timeout = null;
    }
  }

  filterAllHosts = (hosts, selectedLabel) => {
    const { filterHosts } = helpers;

    return filterHosts(hosts, selectedLabel);
  }

  sortHosts = (hosts) => {
    return sortBy(hosts, (h) => { return h.hostname; });
  }

  toggleAddHostModal = () => {
    const { showAddHostModal } = this.state;
    this.setState({ showAddHostModal: !showAddHostModal });
    return false;
  }

  toggleDeleteHostModal = (selectedHost) => {
    return () => {
      const { showDeleteHostModal } = this.state;

      this.setState({
        selectedHost,
        showDeleteHostModal: !showDeleteHostModal,
      });

      return false;
    };
  }

  toggleDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;

    this.setState({ showDeleteLabelModal: !showDeleteLabelModal });
    return false;
  }

  toggleEditLabel = () => {
    const { isEditLabel } = this.state;

    this.setState({ isEditLabel: !isEditLabel });

    return false;
  }

  renderAddHostModal = () => {
    const { toggleAddHostModal } = this;
    const { showAddHostModal } = this.state;
    const { enrollSecret, config } = this.props;

    if (!showAddHostModal) {
      return false;
    }

    return (
      <Modal
        title="Add New Host"
        onExit={toggleAddHostModal}
        className={`${baseClass}__invite-modal`}
      >
        <AddHostModal
          onReturnToApp={toggleAddHostModal}
          enrollSecret={enrollSecret}
          config={config}
        />
      </Modal>
    );
  }

  renderDeleteHostModal = () => {
    const { showDeleteHostModal, selectedHost } = this.state;
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
        <p>This action will delete the host <strong>{selectedHost.hostname}</strong> from your Fleet instance.</p>
        <p>If the host comes back online it will automatically re-enroll. To prevent the host from re-enrolling please disable or uninstall osquery on the host.</p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={onDestroyHost} variant="alert">Delete</Button>
          <Button onClick={toggleDeleteHostModal(null)} variant="inverse">Cancel</Button>
        </div>
      </Modal>
    );
  }

  renderDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;
    const { toggleDeleteLabelModal, onDeleteLabel } = this;

    if (!showDeleteLabelModal) {
      return false;
    }

    return (
      <Modal
        title="Delete Label"
        onExit={toggleDeleteLabelModal}
        className={`${baseClass}_delete-label__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={onDeleteLabel} variant="alert">Delete</Button>
          <Button onClick={toggleDeleteLabelModal} variant="inverse">Cancel</Button>
        </div>
      </Modal>
    );
  }

  renderDeleteButton = () => {
    const { toggleDeleteLabelModal, toggleEditLabel } = this;
    const { selectedLabel: { type } } = this.props;

    if (type !== 'custom') {
      return false;
    }

    return (
      <div className={`${baseClass}__delete-label`}>
        <Button onClick={toggleEditLabel} variant="inverse">Edit</Button>
        <Button onClick={toggleDeleteLabelModal} variant="alert">Delete</Button>
      </div>
    );
  }

  renderIcon = () => {
    const { selectedLabel } = this.props;

    if (platformIconClass(selectedLabel.display_text)) {
      return <PlatformIcon name={platformIconClass(selectedLabel.display_text)} title={platformIconClass(selectedLabel.display_text)} />;
    }

    return <Icon name={iconClassForLabel(selectedLabel)} />;
  }

  renderQuery = () => {
    const { selectedLabel } = this.props;
    const { slug, label_type: labelType, label_membership_type: membershipType, query } = selectedLabel;

    if (membershipType === 'manual' && labelType !== 'builtin') {
      return (
        <h4 title="Manage manual labels with fleetctl">Manually managed</h4>
      );
    }

    if (!query || slug === 'all-hosts') {
      return false;
    }

    return (
      <AceEditor
        editorProps={{ $blockScrolling: Infinity }}
        mode="kolide"
        minLines={1}
        maxLines={20}
        name="label-header"
        readOnly
        setOptions={{ wrap: true }}
        showGutter={false}
        showPrintMargin={false}
        theme="kolide"
        value={query}
        width="100%"
        fontSize={14}
      />
    );
  }

  renderHeader = () => {
    const { renderIcon, renderQuery, renderDeleteButton } = this;
    const { display, isAddLabel, selectedLabel, statusLabels } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { count, description, display_text: displayText, statusLabelKey, type } = selectedLabel;
    const { onToggleDisplay } = this;

    const hostCount = type === 'status' ? statusLabels[`${statusLabelKey}`] : count;
    const hostsTotalDisplay = hostCount === 1 ? '1 Host Total' : `${hostCount} Hosts Total`;
    const defaultDescription = 'No description available.';

    return (
      <div className={`${baseClass}__header`}>
        {renderDeleteButton()}
        <h1 className={`${baseClass}__title`}>
          {renderIcon()}
          <span>{displayText}</span>
        </h1>
        {renderQuery()}
        <div className={`${baseClass}__description`}>
          <h2>Description</h2>
          <p>{description || <em>{defaultDescription}</em>}</p>
        </div>
        <div className={`${baseClass}__topper`}>
          <p className={`${baseClass}__host-count`}>{hostsTotalDisplay}</p>
          <a
            onClick={onToggleDisplay}
            className={classNames(`${baseClass}__toggle-view`, {
              [`${baseClass}__toggle-view--active`]: display === 'List',
            })}
            data-value="List"
          >{<svg viewBox="0 0 24 24" className="kolidecon"><path d="M9 4h11a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1zm0 7h11a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1v-1a1 1 0 0 1 1-1zm0 7h11a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1v-1a1 1 0 0 1 1-1zm-4.5 3a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm0-7a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm0-7a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3z" fillRule="evenodd" /></svg>}
          </a>
          <a
            onClick={onToggleDisplay}
            className={classNames(`${baseClass}__toggle-view`, {
              [`${baseClass}__toggle-view--active`]: display === 'Grid',
            })}
            data-value="Grid"
          >{<svg viewBox="0 0 24 24" className="kolidecon"><path d="M5 15v4h4v-4H5zm-1-2h6a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1zm11 2v4h4v-4h-4zm-1-2h6a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-6a1 1 0 0 1-1-1v-6a1 1 0 0 1 1-1zm1-8v4h4V5h-4zm-1-2h6a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-6a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1zM5 5v4h4V5H5zM4 3h6a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1z" fillRule="nonzero" /></svg>}
          </a>
        </div>
      </div>
    );
  }

  renderForm = () => {
    const { isAddLabel, labelErrors, selectedLabel } = this.props;
    const { isEditLabel } = this.state;
    const {
      onCancelAddLabel,
      onEditLabel,
      onOsqueryTableSelect,
      onSaveAddLabel,
      toggleEditLabel,
    } = this;

    if (isAddLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            onCancel={onCancelAddLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onSaveAddLabel}
            serverErrors={labelErrors}
          />
        </div>
      );
    }

    if (isEditLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            formData={selectedLabel}
            onCancel={toggleEditLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onEditLabel}
            isEdit
            serverErrors={labelErrors}
          />
        </div>
      );
    }

    return false;
  }

  renderSidePanel = () => {
    let SidePanel;
    const {
      isAddLabel,
      labels,
      selectedFilter,
      selectedOsqueryTable,
      statusLabels,
    } = this.props;
    const { onAddHostClick, onAddLabelClick, onLabelClick, onOsqueryTableSelect } = this;

    if (isAddLabel) {
      SidePanel = (
        <QuerySidePanel
          key="query-side-panel"
          onOsqueryTableSelect={onOsqueryTableSelect}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      );
    } else {
      SidePanel = (
        <HostSidePanel
          key="hosts-side-panel"
          labels={labels}
          onAddHostClick={onAddHostClick}
          onAddLabelClick={onAddLabelClick}
          onLabelClick={onLabelClick}
          selectedFilter={selectedFilter}
          statusLabels={statusLabels}
        />
      );
    }

    return SidePanel;
  }

  render () {
    const {
      onQueryHost,
      onPaginationChange,
      renderForm,
      renderHeader,
      renderSidePanel,
      renderAddHostModal,
      renderDeleteHostModal,
      renderDeleteLabelModal,
      toggleAddHostModal,
      toggleDeleteHostModal,
    } = this;
    const {
      page,
      perPage,
      hosts,
      display,
      isAddLabel,
      loadingLabels,
      loadingHosts,
      selectedLabel,
      statusLabels,
    } = this.props;
    const { isEditLabel } = this.state;

    const sortedHosts = this.sortHosts(hosts);


    let hostCount = 0;
    if (hostCount === 0) {
      switch (selectedLabel ? selectedLabel.id : '') {
        case 'all-hosts':
          hostCount = statusLabels.total_count;
          break;
        case 'new':
          hostCount = statusLabels.new_count;
          break;
        case 'online':
          hostCount = statusLabels.online_count;
          break;
        case 'offline':
          hostCount = statusLabels.offline_count;
          break;
        case 'mia':
          hostCount = statusLabels.mia_count;
          break;
        default:
          hostCount = selectedLabel ? selectedLabel.count : 0;
          break;
      }
    }

    return (
      <div className="has-sidebar">
        {renderForm()}

        {!isAddLabel && !isEditLabel &&
          <div className={`${baseClass} body-wrap`}>
            {renderHeader()}
            <div className={`${baseClass}__list ${baseClass}__list--${display.toLowerCase()}`}>
              <HostContainer
                hosts={sortedHosts}
                selectedLabel={selectedLabel}
                displayType={display}
                loadingHosts={loadingHosts}
                toggleAddHostModal={toggleAddHostModal}
                toggleDeleteHostModal={toggleDeleteHostModal}
                onQueryHost={onQueryHost}
              />
              {!loadingHosts && <HostPagination
                allHostCount={hostCount}
                currentPage={page}
                hostsPerPage={perPage}
                onPaginationChange={onPaginationChange}
              />}
            </div>
          </div>
        }

        {!loadingLabels && renderSidePanel()}
        {renderAddHostModal()}
        {renderDeleteHostModal()}
        {renderDeleteLabelModal()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location, params }) => {
  const { active_label: activeLabel, label_id: labelID } = params;
  const activeLabelSlug = activeLabel || 'all-hosts';
  const selectedFilter = labelID ? `labels/${labelID}` : activeLabelSlug;

  const { display, status_labels: statusLabels, page, perPage } = state.components.ManageHostsPage;
  const { entities: hosts } = entityGetter(state).get('hosts');
  const labelEntities = entityGetter(state).get('labels');
  const { entities: labels } = labelEntities;
  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const selectedLabel = labelEntities.findBy(
    { slug: selectedFilter },
    { ignoreCase: true },
  );
  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const { loading: loadingHosts } = state.entities.hosts;
  const enrollSecret = state.app.enrollSecret;
  const config = state.app.config;

  return {
    selectedFilter,
    page,
    perPage,
    display,
    hosts,
    isAddLabel,
    labelErrors,
    labels,
    loadingHosts,
    loadingLabels,
    enrollSecret,
    selectedLabel,
    selectedOsqueryTable,
    statusLabels,
    config,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
