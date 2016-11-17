import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { filter } from 'lodash';
import ReactCSSTransitionGroup from 'react-addons-css-transition-group';

import entityGetter from 'redux/utilities/entityGetter';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import labelInterface from 'interfaces/label';
import HostDetails from 'components/hosts/HostDetails';
import hostInterface from 'interfaces/host';
import HostSidePanel from 'components/side_panels/HostSidePanel';
import osqueryTableInterface from 'interfaces/osquery_table';
import QueryComposer from 'components/queries/QueryComposer';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import { setSelectedLabel } from 'redux/nodes/components/ManageHostsPage/actions';
import { showRightSidePanel, removeRightSidePanel } from 'redux/nodes/app/actions';
import validateQuery from 'components/forms/validators/validate_query';

export class ManageHostsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    hosts: PropTypes.arrayOf(hostInterface),
    labels: PropTypes.arrayOf(labelInterface),
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
  };

  constructor (props) {
    super(props);

    this.state = {
      isAddLabel: false,
      labelQueryText: '',
    };
  }

  componentWillMount () {
    const {
      dispatch,
      hosts,
      labels,
      selectedLabel,
    } = this.props;
    const allHostLabel = filter(labels, { type: 'all' })[0];

    dispatch(showRightSidePanel);

    if (!hosts.length) {
      dispatch(hostActions.loadAll());
    }

    if (!labels.length) {
      dispatch(labelActions.loadAll());
    }

    if (!selectedLabel) {
      dispatch(setSelectedLabel(allHostLabel));
    }

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { dispatch, labels, selectedLabel } = nextProps;
    const allHostLabel = filter(labels, { type: 'all' })[0];

    if (!selectedLabel && !!allHostLabel) {
      dispatch(setSelectedLabel(allHostLabel));
    }

    return false;
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(removeRightSidePanel);
  }

  onCancelAddLabel = () => {
    this.setState({ isAddLabel: false });

    return false;
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();

    this.setState({
      isAddLabel: true,
    });

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

      dispatch(setSelectedLabel(selectedLabel));

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
    const { labelQueryText } = this.state;

    const { error } = validateQuery(labelQueryText);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    return dispatch(labelActions.create(formData))
      .then(() => {
        this.setState({ isAddLabel: false });

        return false;
      });
  }

  onTextEditorInputChange = (labelQueryText) => {
    this.setState({ labelQueryText });

    return false;
  }

  renderHeader = () => {
    const { selectedLabel } = this.props;
    const { isAddLabel } = this.state;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { count, description, display_text: displayText, query } = selectedLabel;

    return (
      <div>
        <i className="kolidecon-label" />
        <span>{displayText}</span>
        <AceEditor
          editorProps={{ $blockScrolling: Infinity }}
          mode="kolide"
          minLines={2}
          maxLines={4}
          name="label-header"
          readOnly
          setOptions={{ wrap: true }}
          showGutter={false}
          showPrintMargin={false}
          theme="kolide"
          value={query}
          width="100%"
        />
        <p>Description</p>
        <p>{description}</p>
        <p>{count} Hosts Total</p>
      </div>
    );
  }

  renderBody = () => {
    const { hosts } = this.props;
    const { isAddLabel, labelQueryText } = this.state;
    const {
      onCancelAddLabel,
      onHostDetailActionClick,
      onSaveAddLabel,
      onTextEditorInputChange,
    } = this;

    if (isAddLabel) {
      return (
        <QueryComposer
          key="query-composer"
          onCancel={onCancelAddLabel}
          onSave={onSaveAddLabel}
          onTextEditorInputChange={onTextEditorInputChange}
          queryType="label"
          queryText={labelQueryText}
        />
      );
    }

    return hosts.map((host) => {
      return (
        <HostDetails
          host={host}
          key={host.hostname}
          onDisableClick={onHostDetailActionClick('disable')}
          onQueryClick={onHostDetailActionClick('query')}
        />
      );
    });
  }

  renderSidePanel = () => {
    let SidePanel;
    const { isAddLabel } = this.state;
    const {
      labels,
      selectedLabel,
      selectedOsqueryTable,
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
        />
      );
    }

    return (
      <ReactCSSTransitionGroup
        transitionName="hosts-page-side-panel"
        transitionEnterTimeout={500}
        transitionLeaveTimeout={0}
      >
        {SidePanel}
      </ReactCSSTransitionGroup>
    );
  }

  render () {
    const { renderBody, renderHeader, renderSidePanel } = this;

    return (
      <div className="manage-hosts">
        {renderHeader()}
        {renderBody()}
        {renderSidePanel()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { entities: hosts } = entityGetter(state).get('hosts');
  const { entities: labels } = entityGetter(state).get('labels');
  const { selectedLabel } = state.components.ManageHostsPage;
  const { selectedOsqueryTable } = state.components.QueryPages;

  return {
    hosts,
    labels,
    selectedLabel,
    selectedOsqueryTable,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
