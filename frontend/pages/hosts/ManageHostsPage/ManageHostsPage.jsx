import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import AceEditor from "react-ace";
import { connect } from "react-redux";
import { push } from "react-router-redux";
import { sortBy } from "lodash";

import AddHostModal from "components/hosts/AddHostModal";
import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import HostSidePanel from "components/side_panels/HostSidePanel";
import LabelForm from "components/forms/LabelForm";
import Modal from "components/modals/Modal";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import labelInterface from "interfaces/label";
import osqueryTableInterface from "interfaces/osquery_table";
import statusLabelsInterface from "interfaces/status_labels";
import enrollSecretInterface from "interfaces/enroll_secret";
import { selectOsqueryTable } from "redux/nodes/components/QueryPages/actions";
import labelActions from "redux/nodes/entities/labels/actions";
import entityGetter from "redux/utilities/entityGetter";
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";
import PATHS from "router/paths";
import deepDifference from "utilities/deep_difference";
import HostContainer from "./components/HostContainer";

const NEW_LABEL_HASH = "#new_label";
const EDIT_LABEL_HASH = "#edit_label";
const baseClass = "manage-hosts";

export class ManageHostsPage extends PureComponent {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func,
    isAddLabel: PropTypes.bool,
    isEditLabel: PropTypes.bool,
    labelErrors: PropTypes.shape({
      base: PropTypes.string,
    }),
    labels: PropTypes.arrayOf(labelInterface),
    loadingLabels: PropTypes.bool.isRequired,
    enrollSecret: enrollSecretInterface,
    selectedFilter: PropTypes.string,
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
  };

  static defaultProps = {
    loadingLabels: false,
  };

  constructor(props) {
    super(props);

    this.state = {
      labelQueryText: "",
      pagedHosts: [],
      showAddHostModal: false,
      selectedHost: null,
      showDeleteLabelModal: false,
      showHostContainerSpinner: false,
    };
  }

  componentDidMount() {
    const { dispatch } = this.props;
    dispatch(getLabels());
  }

  componentWillUnmount() {
    this.clearHostUpdates();
    return false;
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();

    const { dispatch, selectedFilter } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}${NEW_LABEL_HASH}`));

    return false;
  };

  onEditLabelClick = (evt) => {
    evt.preventDefault();

    const { dispatch, selectedFilter } = this.props;
    console.log(selectedFilter);
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}${EDIT_LABEL_HASH}`));

    return false;
  };

  onCancelAddLabel = () => {
    const { dispatch, selectedFilter } = this.props;

    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));

    return false;
  };

  onCancelEditLabel = () => {
    const { dispatch, selectedFilter } = this.props;

    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));

    return false;
  };

  onAddHostClick = (evt) => {
    evt.preventDefault();

    const { toggleAddHostModal } = this;
    toggleAddHostModal();

    return false;
  };

  onEditLabel = (formData) => {
    const { dispatch, selectedLabel, selectedFilter } = this.props;
    const updateAttrs = deepDifference(formData, selectedLabel);

    return dispatch(labelActions.update(selectedLabel, updateAttrs))
      .then(() => {
        dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));

        return false;
      })
      .catch(() => false);
  };

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();
      const { dispatch } = this.props;
      const { MANAGE_HOSTS } = PATHS;
      const { slug, type } = selectedLabel;
      const nextLocation =
        type === "all" ? MANAGE_HOSTS : `${MANAGE_HOSTS}/${slug}`;
      dispatch(push(nextLocation));
    };
  };

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  };

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData)).then(() => {
      dispatch(push(PATHS.MANAGE_HOSTS));

      return false;
    });
  };

  onDeleteLabel = () => {
    const { toggleDeleteLabelModal } = this;
    const { dispatch, selectedLabel } = this.props;
    const { MANAGE_HOSTS } = PATHS;

    return dispatch(labelActions.destroy(selectedLabel)).then(() => {
      toggleDeleteLabelModal();
      dispatch(push(MANAGE_HOSTS));
      return false;
    });
  };

  clearHostUpdates() {
    if (this.timeout) {
      global.window.clearTimeout(this.timeout);
      this.timeout = null;
    }
  }

  sortHosts = (hosts) => {
    return sortBy(hosts, (h) => {
      return h.hostname;
    });
  };

  toggleAddHostModal = () => {
    const { showAddHostModal } = this.state;
    this.setState({ showAddHostModal: !showAddHostModal });
    return false;
  };

  toggleDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;

    this.setState({ showDeleteLabelModal: !showDeleteLabelModal });
    return false;
  };

  renderAddHostModal = () => {
    const { toggleAddHostModal } = this;
    const { showAddHostModal } = this.state;
    const { enrollSecret, config } = this.props;

    if (!showAddHostModal) {
      return false;
    }

    return (
      <Modal
        title="New host"
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
  };

  renderDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;
    const { toggleDeleteLabelModal, onDeleteLabel } = this;

    if (!showDeleteLabelModal) {
      return false;
    }

    return (
      <Modal
        title="Delete label"
        onExit={toggleDeleteLabelModal}
        className={`${baseClass}_delete-label__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={toggleDeleteLabelModal} variant="inverse">
            Cancel
          </Button>
          <Button onClick={onDeleteLabel} variant="alert">
            Delete
          </Button>
        </div>
      </Modal>
    );
  };

  renderDeleteButton = () => {
    const { toggleDeleteLabelModal, onEditLabelClick } = this;
    const {
      selectedLabel: { type },
    } = this.props;

    if (type !== "custom") {
      return false;
    }

    return (
      <div className={`${baseClass}__label-actions`}>
        <Button onClick={onEditLabelClick} variant="inverse">
          Edit
        </Button>
        <Button onClick={toggleDeleteLabelModal} variant="inverse">
          Delete
        </Button>
      </div>
    );
  };

  renderQuery = () => {
    const { selectedLabel } = this.props;
    const {
      slug,
      label_type: labelType,
      label_membership_type: membershipType,
      query,
    } = selectedLabel;

    if (membershipType === "manual" && labelType !== "builtin") {
      return (
        <h4 title="Manage manual labels with fleetctl">Manually managed</h4>
      );
    }

    if (!query || slug === "all-hosts") {
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
  };

  renderHeader = () => {
    const { renderDeleteButton } = this;
    const { isAddLabel, selectedLabel } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { description, display_text: displayText } = selectedLabel;

    const defaultDescription = "No description available.";

    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          <h1 className={`${baseClass}__title`}>
            <span>{displayText}</span>
          </h1>
          <div className={`${baseClass}__description`}>
            <p>{description || <em>{defaultDescription}</em>}</p>
          </div>
        </div>
        {renderDeleteButton()}
      </div>
    );
  };

  renderForm = () => {
    const { isAddLabel, isEditLabel, labelErrors, selectedLabel } = this.props;
    const {
      onCancelAddLabel,
      onCancelEditLabel,
      onEditLabel,
      onOsqueryTableSelect,
      onSaveAddLabel,
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
            onCancel={onCancelEditLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onEditLabel}
            isEdit
            serverErrors={labelErrors}
          />
        </div>
      );
    }

    return false;
  };

  renderSidePanel = () => {
    let SidePanel;
    const {
      isAddLabel,
      labels,
      selectedFilter,
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
          selectedFilter={selectedFilter}
          statusLabels={statusLabels}
        />
      );
    }

    return SidePanel;
  };

  render() {
    const {
      renderForm,
      renderHeader,
      renderSidePanel,
      renderAddHostModal,
      renderDeleteLabelModal,
      renderQuery,
    } = this;
    const {
      isAddLabel,
      isEditLabel,
      loadingLabels,
      selectedLabel,
      selectedFilter,
    } = this.props;

    const { onAddHostClick } = this;

    return (
      <div className="has-sidebar">
        {renderForm()}

        {!isAddLabel && !isEditLabel && (
          <div className={`${baseClass} body-wrap`}>
            <div className="header-wrap">
              {renderHeader()}
              <Button
                onClick={onAddHostClick}
                className={`${baseClass}__add-hosts button button--brand`}
              >
                <span>Add new host</span>
              </Button>
            </div>
            {selectedLabel && renderQuery()}
            <div className={`${baseClass}__list`}>
              <HostContainer
                selectedFilter={selectedFilter}
                selectedLabel={selectedLabel}
              />
            </div>
          </div>
        )}

        {!loadingLabels && renderSidePanel()}
        {renderAddHostModal()}
        {renderDeleteLabelModal()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location, params }) => {
  const { active_label: activeLabel, label_id: labelID } = params;
  const activeLabelSlug = activeLabel || "all-hosts";
  const selectedFilter = labelID ? `labels/${labelID}` : activeLabelSlug;

  const { status_labels: statusLabels } = state.components.ManageHostsPage;
  const labelEntities = entityGetter(state).get("labels");
  const { entities: labels } = labelEntities;
  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;
  const selectedLabel = labelEntities.findBy(
    { slug: selectedFilter },
    { ignoreCase: true }
  );
  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const enrollSecret = state.app.enrollSecret;
  const config = state.app.config;

  return {
    selectedFilter,
    isAddLabel,
    isEditLabel,
    labelErrors,
    labels,
    loadingLabels,
    enrollSecret,
    selectedLabel,
    selectedOsqueryTable,
    statusLabels,
    config,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
