import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import ReactCSSTransitionGroup from 'react-addons-css-transition-group';

import entityGetter from 'redux/utilities/entityGetter';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import labelInterface from 'interfaces/label';
import HostDetails from 'components/hosts/HostDetails';
import hostInterface from 'interfaces/host';
import HostSidePanel from 'components/side_panels/HostSidePanel';
import HostsTable from 'components/hosts/HostsTable';
import Icon from 'components/Icon';
import osqueryTableInterface from 'interfaces/osquery_table';
import paths from 'router/paths';
import QueryComposer from 'components/queries/QueryComposer';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import Rocker from 'components/buttons/Rocker';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import { setDisplay } from 'redux/nodes/components/ManageHostsPage/actions';
import { showRightSidePanel, removeRightSidePanel } from 'redux/nodes/app/actions';
import validateQuery from 'components/forms/validators/validate_query';

const NEW_LABEL_HASH = '#new_label';

export class ManageHostsPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    display: PropTypes.oneOf(['Grid', 'List']),
    hosts: PropTypes.arrayOf(hostInterface),
    isAddLabel: PropTypes.bool,
    labels: PropTypes.arrayOf(labelInterface),
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
  };

  static defaultProps = {
    display: 'Grid',
  };

  constructor (props) {
    super(props);

    this.state = {
      labelQueryText: '',
    };
  }

  componentWillMount () {
    const {
      dispatch,
      hosts,
      labels,
    } = this.props;

    dispatch(showRightSidePanel);

    if (!hosts.length) {
      dispatch(hostActions.loadAll());
    }

    if (!labels.length) {
      dispatch(labelActions.loadAll());
    }

    return false;
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(removeRightSidePanel);
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
    const { labelQueryText } = this.state;

    const { error } = validateQuery(labelQueryText);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    return dispatch(labelActions.create(formData))
      .then(() => {
        this.setState({ labelQueryText: '' });
        dispatch(push('/hosts/manage'));

        return false;
      });
  }

  onTextEditorInputChange = (labelQueryText) => {
    this.setState({ labelQueryText });

    return false;
  }

  onToggleDisplay = () => {
    const { dispatch, display } = this.props;
    const newDisplay = display === 'Grid' ? 'List' : 'Grid';

    dispatch(setDisplay(newDisplay));

    return false;
  }

  renderHeader = () => {
    const { display, isAddLabel, selectedLabel } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { count, description, display_text: displayText, query } = selectedLabel;
    const { onToggleDisplay } = this;
    const buttonOptions = {
      aIcon: 'grid-select',
      aText: 'Grid',
      bIcon: 'list-select',
      bText: 'List',
    };

    return (
      <div>
        <Icon name="label" />
        <span>{displayText}</span>
        { query &&
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
        }
        <p>Description</p>
        <p>{description}</p>
        <p>{count} Hosts Total</p>
        <div>
          <Rocker
            handleChange={onToggleDisplay}
            name="host-display-toggle"
            options={buttonOptions}
            value={display}
          />
        </div>
      </div>
    );
  }

  renderHosts = () => {
    const { display, hosts, isAddLabel } = this.props;
    const { onHostDetailActionClick } = this;

    if (isAddLabel) {
      return false;
    }

    if (display === 'Grid') {
      return hosts.map((host) => {
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

    return <HostsTable hosts={hosts} />;
  }


  renderForm = () => {
    const { isAddLabel } = this.props;
    const { labelQueryText } = this.state;
    const {
      onCancelAddLabel,
      onSaveAddLabel,
      onTextEditorInputChange,
    } = this;

    if (isAddLabel) {
      return (
        <QueryComposer
          key="query-composer"
          onFormCancel={onCancelAddLabel}
          onSave={onSaveAddLabel}
          onTextEditorInputChange={onTextEditorInputChange}
          queryType="label"
          queryText={labelQueryText}
        />
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
    const { renderForm, renderHeader, renderHosts, renderSidePanel } = this;

    return (
      <div className="manage-hosts body-wrap">
        {renderHeader()}
        {renderForm()}
        {renderHosts()}
        {renderSidePanel()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location, params }) => {
  const activeLabelSlug = params.active_label || 'all-hosts';
  const { display } = state.components.ManageHostsPage;
  const { entities: hosts } = entityGetter(state).get('hosts');
  const labelEntities = entityGetter(state).get('labels');
  const { entities: labels } = labelEntities;
  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const selectedLabel = labelEntities.findBy(
    { slug: activeLabelSlug },
    { ignoreCase: true },
  );
  const { selectedOsqueryTable } = state.components.QueryPages;

  return {
    display,
    hosts,
    isAddLabel,
    labels,
    selectedLabel,
    selectedOsqueryTable,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
