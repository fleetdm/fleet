import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';

import debounce from 'utilities/debounce';
import entityGetter from 'redux/utilities/entityGetter';
import QueryComposer from 'components/queries/QueryComposer';
import osqueryTableInterface from 'interfaces/osquery_table';
import queryActions from 'redux/nodes/entities/queries/actions';
import queryInterface from 'interfaces/query';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { selectOsqueryTable, setQueryText, setSelectedTargets, setSelectedTargetsQuery } from 'redux/nodes/components/QueryPages/actions';
import targetInterface from 'interfaces/target';
import validateQuery from 'components/forms/validators/validate_query';

class QueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    query: queryInterface,
    queryText: PropTypes.string,
    selectedOsqueryTable: osqueryTableInterface,
    selectedTargets: PropTypes.arrayOf(targetInterface),
  };

  constructor (props) {
    super(props);

    this.state = {
      targetsCount: 0,
    };
  }

  componentWillMount () {
    const { dispatch, query } = this.props;

    if (query) {
      dispatch(setQueryText(query.query));
    }

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

  onFetchTargets = (query, targetResponse) => {
    const { dispatch } = this.props;
    const {
      targets_count: targetsCount,
    } = targetResponse;

    dispatch(setSelectedTargetsQuery(query));
    this.setState({ targetsCount });

    return false;
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

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

  render () {
    const {
      onFetchTargets,
      onOsqueryTableSelect,
      onRunQuery,
      onSaveQueryFormSubmit,
      onTargetSelect,
      onTextEditorInputChange,
      onUpdateQuery,
    } = this;
    const { targetsCount } = this.state;
    const {
      query,
      queryText,
      selectedOsqueryTable,
      selectedTargets,
    } = this.props;

    return (
      <div className="has-sidebar">
        <QueryComposer
          onFetchTargets={onFetchTargets}
          onOsqueryTableSelect={onOsqueryTableSelect}
          onRunQuery={onRunQuery}
          onSave={onSaveQueryFormSubmit}
          onTargetSelect={onTargetSelect}
          onTextEditorInputChange={onTextEditorInputChange}
          onUpdate={onUpdateQuery}
          query={query}
          selectedTargets={selectedTargets}
          targetsCount={targetsCount}
          selectedOsqueryTable={selectedOsqueryTable}
          queryText={queryText}
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
