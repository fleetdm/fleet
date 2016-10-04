import React, { Component } from 'react';
import NewQuery from '../../../components/Queries/NewQuery';

class NewQueryPage extends Component {
  constructor (props) {
    super(props);

    this.state = {
      selectedOsqueryTable: 'users',
      textEditorText: 'SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid',
    };
  }

  onNewQueryFormSubmit = (formData) => {
    const { textEditorText } = this.state;
    const data = {
      queryText: textEditorText,
      ...formData,
    };

    console.log('New Query Form submitted', data);

    return false;
  }

  onOsqueryTableSelect = (selectedOsqueryTable) => {
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
      </div>
    );
  }
}

export default NewQueryPage;
