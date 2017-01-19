import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { filter, orderBy, sortBy } from 'lodash';
import moment from 'moment';
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
import Rocker from 'components/buttons/Rocker';
import Button from 'components/buttons/Button';
import Modal from 'components/modals/Modal';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';
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
      showDeleteModal: false,
    };
  }

  componentWillMount () {
    const { dispatch } = this.props;

    dispatch(hostActions.loadAll());
    dispatch(labelActions.loadAll());
    dispatch(getStatusLabelCounts);

    return false;
  }

  onCancelAddLabel = () => {
    const { dispatch } = this.props;

    dispatch(push('/hosts/manage'));

    return false;
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(push(`/hosts/manage${NEW_LABEL_HASH}`));

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
    const { toggleModal } = this;
    const { dispatch, selectedLabel } = this.props;
    const { MANAGE_HOSTS } = paths;

    return dispatch(labelActions.destroy(selectedLabel))
      .then(() => {
        toggleModal();
        dispatch(push(MANAGE_HOSTS));
        dispatch(renderFlash('success', 'Label successfully deleted'));
        return false;
      });
  }

  toggleModal = () => {
    const { showDeleteModal } = this.state;

    this.setState({ showDeleteModal: !showDeleteModal });
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

  renderModal = () => {
    const { showDeleteModal } = this.state;
    const { toggleModal, onDeleteLabel } = this;

    if (!showDeleteModal) {
      return false;
    }

    return (
      <Modal
        title="Delete Label"
        onExit={toggleModal}
        className={`${baseClass}__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div>
          <Button onClick={toggleModal} variant="inverse">Cancel</Button>
          <Button onClick={onDeleteLabel} variant="alert">Delete</Button>
        </div>
      </Modal>
    );
  }

  renderDeleteButton = () => {
    const { toggleModal } = this;
    const { selectedLabel: { type } } = this.props;

    if (type !== 'custom') {
      return false;
    }

    return (
      <div className={`${baseClass}__delete-label`}>
        <Button onClick={toggleModal} variant="alert">Delete</Button>
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
    const { onHostDetailActionClick, filterHosts, sortHosts, renderNoHosts } = this;

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
            onDisableClick={onHostDetailActionClick('disable')}
            onQueryClick={onHostDetailActionClick('query')}
          />
        );
      });
    }

    return <HostsTable hosts={sortedHosts} />;
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
    const { renderForm, renderHeader, renderHosts, renderSidePanel, renderModal } = this;
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
        {renderModal()}
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

  // TODO: remove this once the API is updated to return new_count
  statusLabels.new_count = filter(hosts, (h) => {
    return moment().diff(h.created_at, 'hours') <= 24;
  }).length;

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
