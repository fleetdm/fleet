import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { find } from 'lodash';

import Kolide from '../../../kolide';
import NewQuery from '../../../components/queries/NewQuery';
import { osqueryTables } from '../../../utilities/osquery_tables';
import QuerySidePanel from '../../../components/side_panels/QuerySidePanel';
import { showRightSidePanel, removeRightSidePanel } from '../../../redux/nodes/app/actions';
import { renderFlash } from '../../../redux/nodes/notifications/actions';

class NewQueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
  };

  componentWillMount () {
    const { dispatch } = this.props;
    const selectedOsqueryTable = find(osqueryTables, { name: 'users' });

    this.state = {
      isLoadingTargets: false,
      moreInfoTarget: null,
      selectedTargetsCount: 0,
      selectedOsqueryTable,
      targets: [],
      textEditorText: 'SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid',
    };

    dispatch(showRightSidePanel);
    this.fetchTargets();

    return false;
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(removeRightSidePanel);

    return false;
  }

  onNewQueryFormSubmit = (formData) => {
    console.log('New Query Form submitted', formData);
  }

  onInvalidQuerySubmit = (errorMessage) => {
    const { dispatch } = this.props;

    dispatch(renderFlash('error', errorMessage));

    return false;
  }

  onOsqueryTableSelect = (tableName) => {
    const selectedOsqueryTable = find(osqueryTables, { name: tableName.toLowerCase() });
    this.setState({ selectedOsqueryTable });

    return false;
  }

  onRemoveMoreInfoTarget = (evt) => {
    evt.preventDefault();

    this.setState({ moreInfoTarget: null });

    return false;
  }

  onTargetSelectMoreInfo = (moreInfoTarget) => {
    return (evt) => {
      evt.preventDefault();

      const { target_type: targetType } = moreInfoTarget;

      if (targetType.toLowerCase() === 'labels') {
        return Kolide.getLabelHosts(moreInfoTarget.id)
          .then((hosts) => {
            console.log('hosts', hosts);
            this.setState({
              moreInfoTarget: {
                ...moreInfoTarget,
                hosts,
              },
            });

            return false;
          });
      }


      this.setState({ moreInfoTarget });

      return false;
    };
  }

  onTextEditorInputChange = (textEditorText) => {
    this.setState({ textEditorText });

    return false;
  }

  fetchTargets = (search) => {
    this.setState({ isLoadingTargets: true });

    return Kolide.getTargets({ search })
      .then((response) => {
        const {
          selected_targets_count: selectedTargetsCount,
          targets,
        } = response;

        this.setState({
          isLoadingTargets: false,
          selectedTargetsCount,
          targets,
        });

        return search;
      })
      .catch((error) => {
        this.setState({ isLoadingTargets: false });

        throw error;
      });
  }

  render () {
    const {
      fetchTargets,
      onNewQueryFormSubmit,
      onInvalidQuerySubmit,
      onOsqueryTableSelect,
      onRemoveMoreInfoTarget,
      onTargetSelectMoreInfo,
      onTextEditorInputChange,
    } = this;
    const {
      isLoadingTargets,
      moreInfoTarget,
      selectedOsqueryTable,
      selectedTargetsCount,
      targets,
      textEditorText } = this.state;

    return (
      <div>
        <NewQuery
          isLoadingTargets={isLoadingTargets}
          moreInfoTarget={moreInfoTarget}
          onNewQueryFormSubmit={onNewQueryFormSubmit}
          onInvalidQuerySubmit={onInvalidQuerySubmit}
          onOsqueryTableSelect={onOsqueryTableSelect}
          onRemoveMoreInfoTarget={onRemoveMoreInfoTarget}
          onTargetSelectInputChange={fetchTargets}
          onTargetSelectMoreInfo={onTargetSelectMoreInfo}
          onTextEditorInputChange={onTextEditorInputChange}
          selectedTargetsCount={selectedTargetsCount}
          selectedOsqueryTable={selectedOsqueryTable}
          targets={targets}
          textEditorText={textEditorText}
        />
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          onTextEditorInputChange={onTextEditorInputChange}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      </div>
    );
  }
}

export default connect()(NewQueryPage);
