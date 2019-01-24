import React, { Component } from 'react';
import PropTypes from 'prop-types';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import FileSaver from 'file-saver';
import { push } from 'react-router-redux';
import { isEqual, orderBy, slice, sortBy } from 'lodash';
import classNames from 'classnames';

import Kolide from 'kolide';
import AddHostModal from 'components/hosts/AddHostModal';
import Button from 'components/buttons/Button';
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
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import { getStatusLabelCounts, setDisplay, silentGetStatusLabelCounts } from 'redux/nodes/components/ManageHostsPage/actions';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';
import entityGetter from 'redux/utilities/entityGetter';
import paths from 'router/paths';
import deepDifference from 'utilities/deep_difference';
import iconClassForLabel from 'utilities/icon_class_for_label';
import platformIconClass from 'utilities/platform_icon_class';
import scrollToTop from 'utilities/scroll_to_top';
import helpers from './helpers';

const NEW_LABEL_HASH = '#new_label';
const baseClass = 'manage-hosts';

export class ManageHostsPage extends Component {
  static propTypes = {
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
    osqueryEnrollSecret: PropTypes.string,
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
  };

  static defaultProps = {
    display: 'Grid',
    loadingHosts: false,
    loadingLabels: false,
  };

  constructor (props) {
    super(props);

    this.state = {
      allHostCount: 0,
      currentPaginationPage: 0,
      hostsLoading: false,
      hostsPerPage: 20,
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

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(hostActions.loadAll());
    this.buildSortedHosts();

    return this.getEntities();
  }

  componentDidMount () {
    const { getEntities } = this;

    this.interval = global.window.setInterval(getEntities, 5000);

    return false;
  }

  shouldComponentUpdate (nextProps, nextState) {
    if (isEqual(nextProps, this.props) && isEqual(nextState, this.state)) {
      return false;
    }

    this.buildSortedHosts(nextProps, nextState);
    return true;
  }

  componentWillUnmount () {
    if (this.interval) {
      global.window.clearInterval(this.interval);

      this.interval = null;
    }

    return false;
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(push(`/hosts/manage${NEW_LABEL_HASH}`));

    return false;
  }

  onCancelAddLabel = () => {
    const { dispatch } = this.props;

    dispatch(push('/hosts/manage'));

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

  onFetchCertificate = () => {
    return Kolide.config.loadCertificate()
      .then((certificate) => {
        const filename = `${global.window.location.host}.pem`;
        const file = new global.window.File([certificate], filename, { type: 'application/x-pem-file' });

        FileSaver.saveAs(file);

        return false;
      });
  }

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { MANAGE_HOSTS } = paths;
      const { slug } = selectedLabel;
      const nextLocation = slug === 'all-hosts' ? MANAGE_HOSTS : `${MANAGE_HOSTS}/${slug}`;

      dispatch(push(nextLocation));

      this.setState({
        currentPaginationPage: 0,
        hostsLoading: true,
      });

      return false;
    };
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  }

  onPaginationChange = (page) => {
    this.setState({
      currentPaginationPage: page - 1,
      hostsLoading: true,
    });

    scrollToTop();

    return true;
  }

  onPerPageChange = (option) => {
    this.setState({
      currentPaginationPage: 0,
      hostsPerPage: Number(option.value),
      hostsLoading: true,
      showHostContainerSpinner: true,
    });

    scrollToTop();

    return true;
  }

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData))
      .then(() => {
        dispatch(push('/hosts/manage'));

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
    const { MANAGE_HOSTS } = paths;

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
      const { NEW_QUERY } = paths;

      dispatch(push({
        pathname: NEW_QUERY,
        query: { host_ids: [host.id] },
      }));

      return false;
    };
  }

  getEntities = () => {
    const { dispatch } = this.props;

    const promises = [
      dispatch(hostActions.silentLoadAll()),
      dispatch(labelActions.silentLoadAll()),
      dispatch(silentGetStatusLabelCounts),
    ];

    Promise.all(promises)
      .catch(() => false);

    return false;
  }

  buildSortedHosts = (nextProps, nextState) => {
    const { filterAllHosts, sortHosts } = this;
    const { currentPaginationPage, hostsPerPage } = nextState || this.state;
    const { hosts, selectedLabel } = this.props;

    const sortedHosts = sortHosts(filterAllHosts(hosts, selectedLabel));

    const fromIndex = currentPaginationPage * hostsPerPage;
    const toIndex = fromIndex + hostsPerPage;

    const pagedHosts = slice(sortedHosts, fromIndex, toIndex);

    this.setState({
      allHostCount: sortedHosts.length,
      hostsLoading: false,
      pagedHosts,
    });
  }

  filterAllHosts = (hosts, selectedLabel) => {
    const { filterHosts } = helpers;

    return filterHosts(hosts, selectedLabel);
  }

  sortHosts = (hosts) => {
    const alphaHosts = sortBy(hosts, (h) => { return h.hostname; });
    const orderedHosts = orderBy(alphaHosts, 'status', 'desc');

    return orderedHosts;
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
    const { onFetchCertificate, toggleAddHostModal } = this;
    const { showAddHostModal } = this.state;
    const { dispatch, osqueryEnrollSecret } = this.props;

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
          dispatch={dispatch}
          onFetchCertificate={onFetchCertificate}
          onReturnToApp={toggleAddHostModal}
          osqueryEnrollSecret={osqueryEnrollSecret}
        />
      </Modal>
    );
  }

  renderDeleteHostModal = () => {
    const { showDeleteHostModal } = this.state;
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
        <p>This action will delete the host from your Fleet instance.</p>
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
        className={`${baseClass}__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div>
          <Button onClick={toggleDeleteLabelModal} variant="inverse">Cancel</Button>
          <Button onClick={onDeleteLabel} variant="alert">Delete</Button>
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
    const { label_type: labelType, query } = selectedLabel;

    if (!query || labelType === 1) {
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
      selectedLabel,
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
          selectedLabel={selectedLabel}
          statusLabels={statusLabels}
        />
      );
    }

    return SidePanel;
  }

  render () {
    const {
      onQueryHost,
      onPerPageChange,
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
    const { display, isAddLabel, loadingLabels, loadingHosts, selectedLabel } = this.props;
    const { allHostCount, currentPaginationPage, hostsPerPage, hostsLoading, isEditLabel, pagedHosts } = this.state;

    return (
      <div className="has-sidebar">
        {renderForm()}

        {!isAddLabel && !isEditLabel &&
          <div className={`${baseClass} body-wrap`}>
            {renderHeader()}
            <div className={`${baseClass}__list ${baseClass}__list--${display.toLowerCase()}`}>
              <HostContainer
                hosts={pagedHosts}
                selectedLabel={selectedLabel}
                displayType={display}
                loadingHosts={hostsLoading || loadingHosts}
                toggleAddHostModal={toggleAddHostModal}
                toggleDeleteHostModal={toggleDeleteHostModal}
                onQueryHost={onQueryHost}
              />
              {!(hostsLoading || loadingHosts) && <HostPagination
                allHostCount={allHostCount}
                currentPage={currentPaginationPage}
                hostsPerPage={hostsPerPage}
                onPaginationChange={onPaginationChange}
                onPerPageChange={onPerPageChange}
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
  const activeLabelSlug = params.active_label || 'all-hosts';
  const { display, status_labels: statusLabels } = state.components.ManageHostsPage;
  const { entities: hosts } = entityGetter(state).get('hosts');
  const labelEntities = entityGetter(state).get('labels');
  const { entities: labels } = labelEntities;
  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const selectedLabel = labelEntities.findBy(
    { slug: activeLabelSlug },
    { ignoreCase: true },
  );
  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const { loading: loadingHosts } = state.entities.hosts;
  const { osquery_enroll_secret: osqueryEnrollSecret } = state.app.config;

  return {
    display,
    hosts,
    isAddLabel,
    labelErrors,
    labels,
    loadingHosts,
    loadingLabels,
    osqueryEnrollSecret,
    selectedLabel,
    selectedOsqueryTable,
    statusLabels,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
