import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { find } from 'lodash';
import NewQuery from '../../../components/Queries/NewQuery';
import QuerySidePanel from '../../../components/Queries/QuerySidePanel';
import { showRightSidePanel, removeRightSidePanel } from '../../../redux/nodes/app/actions';
import { osqueryTables } from '../../../utilities/osquery_tables';

class NewQueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
  };

  componentWillMount () {
    const { dispatch } = this.props;
    const selectedOsqueryTable = find(osqueryTables, { name: 'users' });

    this.state = {
      selectedOsqueryTable,
      textEditorText: 'SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid',
    };

    dispatch(showRightSidePanel);

    return false;
  }

  componentWillUnmount () {
    const { dispatch } = this.props;

    dispatch(removeRightSidePanel);

    return false;
  }

  onNewQueryFormSubmit = (formData) => {
    const { textEditorText } = this.state;
    const data = {
      queryText: textEditorText,
      ...formData,
    };

    console.log('New Query Form submitted', data);
  }

  onOsqueryTableSelect = (tableName) => {
    const selectedOsqueryTable = find(osqueryTables, { name: tableName.toLowerCase() });
    this.setState({ selectedOsqueryTable });

    return false;
  }

  onTextEditorInputChange = (textEditorText) => {
    this.setState({ textEditorText });

    return false;
  }

  render () {
    const { selectedOsqueryTable, textEditorText } = this.state;
    const { onNewQueryFormSubmit, onOsqueryTableSelect, onTextEditorInputChange } = this;

    return (
      <div>
        <NewQuery
          onNewQueryFormSubmit={onNewQueryFormSubmit}
          onOsqueryTableSelect={onOsqueryTableSelect}
          onTextEditorInputChange={onTextEditorInputChange}
          selectedOsqueryTable={selectedOsqueryTable}
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
