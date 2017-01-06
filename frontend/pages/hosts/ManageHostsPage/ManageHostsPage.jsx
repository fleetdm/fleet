import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';

import entityGetter from 'redux/utilities/entityGetter';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';
import labelInterface from 'interfaces/label';
import HostDetails from 'components/hosts/HostDetails';
import hostInterface from 'interfaces/host';
import HostSidePanel from 'components/side_panels/HostSidePanel';
import HostsTable from 'components/hosts/HostsTable';
import Icon from 'components/icons/Icon';
import osqueryTableInterface from 'interfaces/osquery_table';
import paths from 'router/paths';
import QueryForm from 'components/forms/queries/QueryForm';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import Rocker from 'components/buttons/Rocker';
import { selectOsqueryTable } from 'redux/nodes/components/QueryPages/actions';
import { setDisplay } from 'redux/nodes/components/ManageHostsPage/actions';
import iconClassForLabel from 'utilities/icon_class_for_label';

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

    if (!hosts.length) {
      dispatch(hostActions.loadAll());
    }

    if (!labels.length) {
      dispatch(labelActions.loadAll());
    }

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

  renderHeader = () => {
    const { display, isAddLabel, selectedLabel } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { count, description, display_text: displayText, query } = selectedLabel;
    const { onToggleDisplay } = this;
    const buttonOptions = {
      rightIcon: 'grid-select',
      rightText: 'Grid',
      leftIcon: 'list-select',
      leftText: 'List',
    };

    return (
      <div className={`${baseClass}__header`}>
        <h1 className={`${baseClass}__title`}>
          <Icon name={iconClassForLabel(selectedLabel)} />
          <span>{displayText}</span>
        </h1>

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

        <div className={`${baseClass}__description`}>
          <h2>Description</h2>
          <p>{description}</p>
        </div>

        <div className={`${baseClass}__topper`}>
          <p className={`${baseClass}__host-count`}>{count} Hosts Total</p>
          <Rocker
            onChange={onToggleDisplay}
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
    const { isAddLabel, labelErrors } = this.props;
    const {
      onCancelAddLabel,
      onOsqueryTableSelect,
      onSaveAddLabel,
    } = this;
    const queryStub = { description: '', name: '', query: '' };

    if (isAddLabel) {
      return (
        <QueryForm
          key="query-composer"
          onCancel={onCancelAddLabel}
          onOsqueryTableSelect={onOsqueryTableSelect}
          handleSubmit={onSaveAddLabel}
          queryType="label"
          query={queryStub}
          serverErrors={labelErrors}
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

    return SidePanel;
  }

  render () {
    const { renderForm, renderHeader, renderHosts, renderSidePanel } = this;
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
  const labelErrors = state.entities.labels.errors;

  return {
    display,
    hosts,
    isAddLabel,
    labelErrors,
    labels,
    selectedLabel,
    selectedOsqueryTable,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
