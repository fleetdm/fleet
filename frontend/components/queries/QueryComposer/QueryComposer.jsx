import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import 'brace/mode/sql';
import 'brace/ext/linking';

import QueryForm from 'components/forms/queries/QueryForm';
import queryInterface from 'interfaces/query';
import SelectTargetsDropdown from 'components/forms/fields/SelectTargetsDropdown';
import targetInterface from 'interfaces/target';
import './mode';
import './theme';

const baseClass = 'query-composer';

class QueryComposer extends Component {
  static propTypes = {
    onFetchTargets: PropTypes.func,
    onFormCancel: PropTypes.func,
    onOsqueryTableSelect: PropTypes.func,
    onRunQuery: PropTypes.func,
    onSave: PropTypes.func,
    onTargetSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    onUpdate: PropTypes.func,
    query: queryInterface,
    queryType: PropTypes.string,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    selectedTargetsCount: PropTypes.number,
    queryText: PropTypes.string,
  };

  static defaultProps = {
    queryType: 'query',
    selectedTargetsCount: 0,
  };

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

  renderForm = () => {
    const {
      onFormCancel,
      onRunQuery,
      onSave,
      onUpdate,
      query,
      queryText,
      queryType,
    } = this.props;

    return (
      <QueryForm
        onCancel={onFormCancel}
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
      onFetchTargets,
      onTargetSelect,
      queryType,
      selectedTargets,
      selectedTargetsCount,
    } = this.props;

    if (queryType === 'label') {
      return false;
    }


    return (
      <div>
        <p className={`${baseClass}__target-label`}>
          <span className={`${baseClass}__select-targets`}>Select Targets</span>
          <span className={`${baseClass}__targets-count`}> {selectedTargetsCount} unique {selectedTargetsCount === 1 ? 'host' : 'hosts' }</span>
        </p>
        <SelectTargetsDropdown
          onFetchTargets={onFetchTargets}
          onSelect={onTargetSelect}
          selectedTargets={selectedTargets}
        />
      </div>
    );
  }

  render () {
    const { onTextEditorInputChange, queryText, queryType } = this.props;
    const { onLoad, renderForm, renderTargetsInput } = this;

    return (
      <div className={`${baseClass}__wrapper`}>
        <h1>{queryType === 'label' ? 'New Label Query' : 'New Query'}</h1>
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
