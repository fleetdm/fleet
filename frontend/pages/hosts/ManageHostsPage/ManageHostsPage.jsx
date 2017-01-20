import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { orderBy, sortBy } from 'lodash';
import { push } from 'react-router-redux';

import entityGetter from 'redux/utilities/entityGetter';
import { getStatusLabelCounts, setDisplay } from 'redux/nodes/components/ManageHostsPage/actions';
import helpers from 'pages/hosts/ManageHostsPage/helpers';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import labelInterface from 'interfaces/label';
import HostDetails from 'components/hosts/HostDetails';
import hostInterface from 'interfaces/host';
import HostSidePanel from 'components/side_panels/HostSidePanel';
import HostsTable from 'components/hosts/HostsTable';
import LonelyHost from 'components/hosts/LonelyHost';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import osqueryTableInterface from 'interfaces/osquery_table';
import paths from 'router/paths';
import QueryForm from 'components/forms/queries/QueryForm';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import Rocker from 'components/buttons/Rocker';
import Button from 'components/buttons/Button';
import Modal from 'components/modals/Modal';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import statusLabelsInterface from 'interfaces/status_labels';
import iconClassForLabel from 'utilities/icon_class_for_label';
import platformIconClass from 'utilities/platform_icon_class';

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
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
  };

  static defaultProps = {
    display: 'Grid',
  };

  constructor (props) {
    super(props);

    this.state = {
      labelQueryText: '',
      selectedHost: null,
      showDeleteLabelModal: false,
    };
  }

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(hostActions.loadAll());
    dispatch(labelActions.loadAll());
    dispatch(getStatusLabelCounts);

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

  onDestroyHost = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;
    const { selectedHost } = this.state;

    dispatch(hostActions.destroy(selectedHost))
      .then(() => {
        this.toggleHostModal(null)();

        dispatch(getStatusLabelCounts);
        dispatch(renderFlash('success', `Host "${selectedHost.hostname}" was successfully deleted`));
      });

    return false;
  }

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { MANAGE_HOSTS } = paths;
      const { slug } = selectedLabel;
      const nextLocation = slug === 'all-hosts' ? MANAGE_HOSTS : `${MANAGE_HOSTS}/${slug}`;

      dispatch(push(nextLocation));

      return false;
    };
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  }

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData))
      .then(() => {
        dispatch(push('/hosts/manage'));

        return false;
      });
  }

  onToggleDisplay = (val) => {
    const { dispatch } = this.props;

    dispatch(setDisplay(val));

    return false;
  }

  onDeleteLabel = () => {
    const { toggleLabelModal } = this;
    const { dispatch, selectedLabel } = this.props;
    const { MANAGE_HOSTS } = paths;

    return dispatch(labelActions.destroy(selectedLabel))
      .then(() => {
        toggleLabelModal();
        dispatch(push(MANAGE_HOSTS));
        dispatch(renderFlash('success', 'Label successfully deleted'));
        return false;
      });
  }

  toggleHostModal = (selectedHost) => {
    return () => {
      const { showDeleteHostModal } = this.state;

      this.setState({
        selectedHost,
        showDeleteHostModal: !showDeleteHostModal,
      });

      return false;
    };
  }

  toggleLabelModal = () => {
    const { showDeleteLabelModal } = this.state;

    this.setState({ showDeleteLabelModal: !showDeleteLabelModal });
    return false;
  }

  filterHosts = () => {
    const { hosts, selectedLabel } = this.props;

    return helpers.filterHosts(hosts, selectedLabel);
  }

  sortHosts = (hosts) => {
    const alphaHosts = sortBy(hosts, (h) => { return h.hostname; });
    const orderedHosts = orderBy(alphaHosts, 'status', 'desc');

    return orderedHosts;
  }

  renderHostModal = () => {
    const { showDeleteHostModal } = this.state;
    const { toggleHostModal, onDestroyHost } = this;

    if (!showDeleteHostModal) {
      return false;
    }

    return (
      <Modal
        title="Delete Host"
        onExit={toggleHostModal(null)}
        className={`${baseClass}__modal`}
      >
        <p>Are you sure you wish to delete this host?</p>
        <div>
          <Button onClick={toggleHostModal(null)} variant="inverse">Cancel</Button>
          <Button onClick={onDestroyHost} variant="alert">Delete</Button>
        </div>
      </Modal>
    );
  }

  renderLabelModal = () => {
    const { showDeleteLabelModal } = this.state;
    const { toggleLabelModal, onDeleteLabel } = this;

    if (!showDeleteLabelModal) {
      return false;
    }

    return (
      <Modal
        title="Delete Label"
        onExit={toggleLabelModal}
        className={`${baseClass}__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div>
          <Button onClick={toggleLabelModal} variant="inverse">Cancel</Button>
          <Button onClick={onDeleteLabel} variant="alert">Delete</Button>
        </div>
      </Modal>
    );
  }

  renderDeleteButton = () => {
    const { toggleLabelModal } = this;
    const { selectedLabel: { type } } = this.props;

    if (type !== 'custom') {
      return false;
    }

    return (
      <div className={`${baseClass}__delete-label`}>
        <Button onClick={toggleLabelModal} variant="alert">Delete</Button>
      </div>
    );
  }

  renderIcon = () => {
    const { selectedLabel } = this.props;

    if (platformIconClass(selectedLabel.display_text)) {
      return <PlatformIcon name={platformIconClass(selectedLabel.display_text)} />;
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
    const buttonOptions = {
      rightIcon: 'grid-select',
      rightText: 'Grid',
      leftIcon: 'list-select',
      leftText: 'List',
    };

    const hostCount = type === 'status' ? statusLabels[`${statusLabelKey}`] : count;
    const hostsTotalDisplay = hostCount === 1 ? '1 Host Total' : `${hostCount} Hosts Total`;

    return (
      <div className={`${baseClass}__header`}>
        {renderDeleteButton()}

        <h1 className={`${baseClass}__title`}>
          {renderIcon()}
          <span>{displayText}</span>
        </h1>

        { renderQuery() }

        {description &&
          <div className={`${baseClass}__description`}>
            <h2>Description</h2>
            <p>{description}</p>
          </div>
        }

        <div className={`${baseClass}__topper`}>
          <p className={`${baseClass}__host-count`}>{hostsTotalDisplay}</p>
          <Rocker
            onChange={onToggleDisplay}
            options={buttonOptions}
            value={display}
          />
        </div>
      </div>
    );
  }

  renderNoHosts = () => {
    const { selectedLabel } = this.props;
    const { type } = selectedLabel || '';
    const isCustom = type === 'custom';

    return (
      <div className={`${baseClass}__no-hosts`}>
        <h1>No matching hosts found.</h1>
        <h2>Where are the missing hosts?</h2>
        <ul>
          {isCustom && <li>Check your SQL query above to confirm there are no mistakes.</li>}
          <li>Check to confirm that your hosts are online.</li>
          <li>Confirm that your expected hosts have osqueryd installed and configured.</li>
        </ul>

        <div className={`${baseClass}__no-hosts-contact`}>
          <p>Still having trouble? Want to talk to a human?</p>
          <p>Contact Kolide Support:</p>
          <p><a href="mailto:support@kolide.co">support@kolide.co</a></p>
        </div>
      </div>
    );
  }

  renderHosts = () => {
    const { display, isAddLabel, selectedLabel } = this.props;
    const { toggleHostModal, filterHosts, sortHosts, renderNoHosts } = this;

    if (isAddLabel) {
      return false;
    }

    const filteredHosts = filterHosts();
    const sortedHosts = sortHosts(filteredHosts);

    if (sortedHosts.length === 0) {
      if (selectedLabel && selectedLabel.type === 'all') {
        return <LonelyHost />;
      }

      return renderNoHosts();
    }

    if (display === 'Grid') {
      return sortedHosts.map((host) => {
        return (
          <HostDetails
            host={host}
            key={`host-${host.id}-details`}
            onDestroyHost={toggleHostModal}
          />
        );
      });
    }

    return <HostsTable hosts={sortedHosts} onDestroyHost={toggleHostModal} />;
  }


  renderForm = () => {
    const { isAddLabel, labelErrors } = this.props;
    const {
      onCancelAddLabel,
      onOsqueryTableSelect,
      onSaveAddLabel,
    } = this;
    const queryStub = { description: '', name: '', query: '' };

    if (isAddLabel) {
      return (
        <div className="body-wrap">
          <QueryForm
            key="query-composer"
            onCancel={onCancelAddLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onSaveAddLabel}
            queryType="label"
            query={queryStub}
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
    const { onAddLabelClick, onLabelClick, onOsqueryTableSelect } = this;

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
    const { renderForm, renderHeader, renderHosts, renderSidePanel, renderHostModal, renderLabelModal } = this;
    const { display, isAddLabel } = this.props;

    return (
      <div className="has-sidebar">
        {renderForm()}
        {!isAddLabel &&
          <div className={`${baseClass} body-wrap`}>
            {renderHeader()}
            <div className={`${baseClass}__list ${baseClass}__list--${display.toLowerCase()}`}>
              {renderHosts()}
            </div>
          </div>
        }

        {renderSidePanel()}
        {renderHostModal()}
        {renderLabelModal()}
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
  const labelErrors = state.entities.labels.errors;

  return {
    display,
    hosts,
    isAddLabel,
    labelErrors,
    labels,
    selectedLabel,
    selectedOsqueryTable,
    statusLabels,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
