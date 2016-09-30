import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import 'brace/ext/linking';
import radium from 'radium';
import './mode';
import './theme';
import componentStyles from './styles';
import SaveQueryForm from '../../forms/queries/SaveQueryForm';
import SaveQuerySection from './SaveQuerySection';
import ThemeDropdown from './ThemeDropdown';

class NewQuery extends Component {
  static propTypes = {
    onOsqueryTableSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    textEditorText: PropTypes.string,
  };

  constructor (props) {
    super(props);

    this.state = {
      saveQuery: false,
      theme: 'kolide',
    };
  }

  onLoad = (editor) => {
    editor.setOptions({
      enableLinking: true,
    });

    editor.on('linkClick', (data) => {
      const { type, value } = data.token;
      const { onOsqueryTableSelect } = this.props;

      if (type === 'osquery-token') {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  }

  onSaveQueryFormSubmit = (formData) => {
    console.log('SaveQueryForm submitted', formData);

    return false;
  }

  onThemeSelect = (evt) => {
    evt.preventDefault();

    this.setState({
      theme: evt.target.value,
    });

    return false;
  }

  onToggleSaveQuery = () => {
    const { saveQuery } = this.state;

    this.setState({
      saveQuery: !saveQuery,
    });

    return false;
  }

  render () {
    const {
      containerStyles,
      selectTargetsHeaderStyles,
      targetsInputStyle,
      titleStyles,
    } = componentStyles;
    const { onTextEditorInputChange, textEditorText } = this.props;
    const { saveQuery, theme } = this.state;
    const {
      onBeforeLoad,
      onLoad,
      onSaveQueryFormSubmit,
      onThemeSelect,
      onToggleSaveQuery,
    } = this;

    return (
      <div style={containerStyles}>
        <p style={titleStyles}>
          New Query Page
        </p>
        <ThemeDropdown onSelectChange={onThemeSelect} theme={theme} />
        <div style={{ marginTop: '20px' }}>
          <AceEditor
            enableBasicAutocompletion
            enableLiveAutocompletion
            editorProps={{ $blockScrolling: Infinity }}
            mode="kolide"
            minLines={4}
            maxLines={4}
            name="query-editor"
            onBeforeLoad={onBeforeLoad}
            onLoad={onLoad}
            onChange={onTextEditorInputChange}
            setOptions={{ enableLinking: true }}
            showGutter
            showPrintMargin={false}
            theme={theme}
            value={textEditorText}
            width="100%"
          />
        </div>
        <div>
          <p style={selectTargetsHeaderStyles}>Select Targets</p>
          <input type="text" style={targetsInputStyle} />
        </div>
        <SaveQuerySection onToggleSaveQuery={onToggleSaveQuery} saveQuery={saveQuery} />
        <SaveQueryForm onSubmit={onSaveQueryFormSubmit} saveQuery={saveQuery} />
      </div>
    );
  }
}

export default radium(NewQuery);
