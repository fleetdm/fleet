import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import 'brace/ext/linking';
import radium from 'radium';

import './mode';
import './theme';
import debounce from '../../../utilities/debounce';
import SaveQueryForm from '../../forms/queries/SaveQueryForm';
import SaveQuerySection from './SaveQuerySection';
import SelectTargetsInput from '../SelectTargetsInput';
import SelectTargetsMenu from '../SelectTargetsMenu';
import targetInterface from '../../../interfaces/target';
import ThemeDropdown from './ThemeDropdown';
import { validateQuery } from './helpers';

const baseClass = 'new-query';

class NewQuery extends Component {
  static propTypes = {
    isLoadingTargets: PropTypes.bool,
    moreInfoTarget: targetInterface,
    onInvalidQuerySubmit: PropTypes.func,
    onNewQueryFormSubmit: PropTypes.func,
    onOsqueryTableSelect: PropTypes.func,
    onRemoveMoreInfoTarget: PropTypes.func,
    onTargetSelectInputChange: PropTypes.func,
    onTargetSelectMoreInfo: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    selectedTargetsCount: PropTypes.number,
    targets: PropTypes.arrayOf(targetInterface),
    textEditorText: PropTypes.string,
  };

  constructor (props) {
    super(props);

    this.state = {
      saveQuery: false,
      selectedTargets: [],
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

  onSaveQueryFormSubmit = debounce((formData) => {
    const {
      onInvalidQuerySubmit,
      onNewQueryFormSubmit,
      textEditorText,
    } = this.props;
    const { selectedTargets } = this.state;

    const { error } = validateQuery(textEditorText);

    if (error) {
      return onInvalidQuerySubmit(error);
    }

    return onNewQueryFormSubmit({
      ...formData,
      query: textEditorText,
      selectedTargets,
    });
  })

  onTargetSelect = (selectedTargets) => {
    this.setState({ selectedTargets });
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
      isLoadingTargets,
      moreInfoTarget,
      onRemoveMoreInfoTarget,
      onTargetSelectInputChange,
      onTargetSelectMoreInfo,
      onTextEditorInputChange,
      selectedTargetsCount,
      targets,
      textEditorText,
    } = this.props;
    const { saveQuery, selectedTargets, theme } = this.state;
    const {
      onLoad,
      onSaveQueryFormSubmit,
      onTargetSelect,
      onThemeSelect,
      onToggleSaveQuery,
    } = this;
    const menuRenderer = SelectTargetsMenu(onTargetSelectMoreInfo, onRemoveMoreInfoTarget, moreInfoTarget);

    return (
      <div className={`${baseClass}__wrapper`}>
        <p className={`${baseClass}__title`}>
          New Query Page
        </p>
        <ThemeDropdown onSelectChange={onThemeSelect} theme={theme} />
        <div className={`${baseClass}__text-editor-wrapper`}>
          <AceEditor
            enableBasicAutocompletion
            enableLiveAutocompletion
            editorProps={{ $blockScrolling: Infinity }}
            mode="kolide"
            minLines={4}
            maxLines={4}
            name="query-editor"
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
          <p>
            <span className={`${baseClass}__select-targets`}>Select Targets</span>
            <span className={`${baseClass}__targets-count`}> {selectedTargetsCount} unique hosts</span>
          </p>
          <SelectTargetsInput
            isLoading={isLoadingTargets}
            menuRenderer={menuRenderer}
            onTargetSelect={onTargetSelect}
            onTargetSelectInputChange={onTargetSelectInputChange}
            selectedTargets={selectedTargets}
            targets={targets}
          />
        </div>
        <SaveQuerySection onToggleSaveQuery={onToggleSaveQuery} saveQuery={saveQuery} />
        <SaveQueryForm onSubmit={onSaveQueryFormSubmit} saveQuery={saveQuery} />
      </div>
    );
  }
}

export default radium(NewQuery);
