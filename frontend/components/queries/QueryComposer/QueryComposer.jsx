import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import 'brace/mode/sql';
import 'brace/ext/linking';

import QueryForm from 'components/forms/queries/QueryForm';
import queryInterface from 'interfaces/query';
import SelectTargets from 'components/forms/fields/SelectTargetsDropdown';
import targetInterface from 'interfaces/target';
import './mode';
import './theme';

const baseClass = 'query-composer';

class QueryComposer extends Component {
  static propTypes = {
    isLoadingTargets: PropTypes.bool,
    moreInfoTarget: targetInterface,
    onCloseTargetSelect: PropTypes.func,
    onCancel: PropTypes.func,
    onOsqueryTableSelect: PropTypes.func,
    onRemoveMoreInfoTarget: PropTypes.func,
    onRunQuery: PropTypes.func,
    onSave: PropTypes.func,
    onTargetSelect: PropTypes.func,
    onTargetSelectInputChange: PropTypes.func,
    onTargetSelectMoreInfo: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    onUpdate: PropTypes.func,
    query: queryInterface,
    queryType: PropTypes.string,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    selectedTargetsCount: PropTypes.number,
    targets: PropTypes.arrayOf(targetInterface),
    queryText: PropTypes.string,
  };

  static defaultProps = {
    queryType: 'query',
    selectedTargetsCount: 0,
  };

  constructor (props) {
    super(props);

    this.state = {
      isSaveQueryForm: false,
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

  onLoadSaveQueryModal = () => {
    this.setState({ isSaveQueryForm: true });

    return false;
  }

  onSaveQueryFormCancel = (evt) => {
    evt.preventDefault();

    this.setState({ isSaveQueryForm: false });

    return false;
  }

  renderForm = () => {
    const {
      onCancel,
      onRunQuery,
      onSave,
      onUpdate,
      query,
      queryText,
      queryType,
    } = this.props;

    return (
      <QueryForm
        onCancel={onCancel}
        onRunQuery={onRunQuery}
        onSave={onSave}
        onUpdate={onUpdate}
        query={query}
        queryType={queryType}
        queryText={queryText}
      />
    );
  }

  renderTargetsInput = () => {
    const {
      isLoadingTargets,
      moreInfoTarget,
      onCloseTargetSelect,
      onRemoveMoreInfoTarget,
      onTargetSelect,
      onTargetSelectInputChange,
      onTargetSelectMoreInfo,
      queryType,
      selectedTargets,
      selectedTargetsCount,
      targets,
    } = this.props;

    if (queryType === 'label') {
      return false;
    }

    const menuRenderer = SelectTargets.Menu(onTargetSelectMoreInfo, onRemoveMoreInfoTarget, moreInfoTarget);

    return (
      <div>
        <p>
          <span className={`${baseClass}__select-targets`}>Select Targets</span>
          <span className={`${baseClass}__targets-count`}> {selectedTargetsCount} unique hosts</span>
        </p>
        <SelectTargets.Input
          isLoading={isLoadingTargets}
          menuRenderer={menuRenderer}
          onCloseTargetSelect={onCloseTargetSelect}
          onTargetSelect={onTargetSelect}
          onTargetSelectInputChange={onTargetSelectInputChange}
          selectedTargets={selectedTargets}
          targets={targets}
        />
      </div>
    );
  }

  render () {
    const { onTextEditorInputChange, queryText } = this.props;
    const { onLoad, renderForm, renderTargetsInput } = this;

    return (
      <div className={`${baseClass}__wrapper`}>
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
            theme="kolide"
            value={queryText}
            width="100%"
          />
        </div>
        {renderTargetsInput()}
        {renderForm()}
      </div>
    );
  }
}

export default QueryComposer;
