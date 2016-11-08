import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';

import debounce from 'utilities/debounce';
import entityGetter from 'redux/utilities/entityGetter';
import Kolide from 'kolide';
import QueryComposer from 'components/queries/QueryComposer';
import osqueryTableInterface from 'interfaces/osquery_table';
import queryActions from 'redux/nodes/entities/queries/actions';
import queryInterface from 'interfaces/query';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { removeRightSidePanel, showRightSidePanel } from 'redux/nodes/app/actions';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { selectOsqueryTable, setQueryText, setSelectedTargets } from 'redux/nodes/components/QueryPages/actions';
import targetInterface from 'interfaces/target';
import { validateQuery } from 'pages/queries/QueryPage/helpers';

class QueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    query: queryInterface,
    queryText: PropTypes.string,
    selectedOsqueryTable: osqueryTableInterface,
    selectedTargets: PropTypes.arrayOf(targetInterface),
  };

  componentWillMount () {
    const { dispatch, query } = this.props;

    if (query) {
      dispatch(setQueryText(query.query));
    }

    this.state = {
      isLoadingTargets: false,
      moreInfoTarget: null,
      selectedTargetsCount: 0,
      targets: [],
    };

    dispatch(showRightSidePanel);
    this.fetchTargets();

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { dispatch, query: newQuery } = nextProps;
    const { query: oldQuery } = this.props;

    if ((!oldQuery && newQuery) || (oldQuery && oldQuery.query !== newQuery.query)) {
      const { query: queryText } = newQuery;

      dispatch(setQueryText(queryText));
    }

    return false;
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(removeRightSidePanel);

    return false;
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  }

  onRemoveMoreInfoTarget = (evt) => {
    evt.preventDefault();

    this.setState({ moreInfoTarget: null });

    return false;
  }

  onRunQuery = debounce((evt) => {
    evt.preventDefault();

    const { dispatch, queryText, selectedTargets } = this.props;
    const { error } = validateQuery(queryText);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    console.log('TODO: dispatch thunk to run query with', { queryText, selectedTargets });

    return false;
  })

  onSaveQueryFormSubmit = debounce((formData) => {
    const { dispatch, queryText } = this.props;
    const { error } = validateQuery(queryText);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    const queryParams = { ...formData, query: queryText };

    return dispatch(queryActions.create(queryParams))
      .then((query) => {
        dispatch(push(`/queries/${query.id}`));
        dispatch(renderFlash('success', 'Query created'));
      })
      .catch((errorResponse) => {
        dispatch(renderFlash('error', errorResponse));
        return false;
      });
  })

  onTargetSelect = (selectedTargets) => {
    const { dispatch } = this.props;

    dispatch(setSelectedTargets(selectedTargets));

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

  onTextEditorInputChange = (queryText) => {
    const { dispatch } = this.props;

    dispatch(setQueryText(queryText));

    return false;
  }

  onUpdateQuery = (formData) => {
    const { dispatch, query } = this.props;

    dispatch(queryActions.update(query, formData))
      .then(() => {
        dispatch(renderFlash('success', 'Query updated!'));
      });

    return false;
  };

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
      onOsqueryTableSelect,
      onRemoveMoreInfoTarget,
      onRunQuery,
      onSaveQueryFormSubmit,
      onTargetSelect,
      onTargetSelectMoreInfo,
      onTextEditorInputChange,
      onUpdateQuery,
    } = this;
    const {
      isLoadingTargets,
      moreInfoTarget,
      selectedTargetsCount,
      targets,
    } = this.state;
    const {
      query,
      queryText,
      selectedOsqueryTable,
      selectedTargets,
    } = this.props;

    return (
      <div>
        <QueryComposer
          isLoadingTargets={isLoadingTargets}
          moreInfoTarget={moreInfoTarget}
          onOsqueryTableSelect={onOsqueryTableSelect}
          onRemoveMoreInfoTarget={onRemoveMoreInfoTarget}
          onRunQuery={onRunQuery}
          onSaveQueryFormSubmit={onSaveQueryFormSubmit}
          onTargetSelect={onTargetSelect}
          onTargetSelectInputChange={fetchTargets}
          onTargetSelectMoreInfo={onTargetSelectMoreInfo}
          onTextEditorInputChange={onTextEditorInputChange}
          onUpdateQuery={onUpdateQuery}
          query={query}
          selectedTargets={selectedTargets}
          selectedTargetsCount={selectedTargetsCount}
          selectedOsqueryTable={selectedOsqueryTable}
          targets={targets}
          textEditorText={queryText}
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

const mapStateToProps = (state, { params }) => {
  const { id: queryID } = params;
  const query = entityGetter(state).get('queries').findBy({ id: queryID });
  const { queryText, selectedOsqueryTable, selectedTargets } = state.components.QueryPages;

  return { query, queryText, selectedOsqueryTable, selectedTargets };
};

export default connect(mapStateToProps)(QueryPage);
