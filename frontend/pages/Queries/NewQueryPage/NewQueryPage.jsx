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
    const { onOsqueryTableSelect, onTextEditorInputChange } = this;

    return (
      <div>
        <NewQuery
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
