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
    onStopQuery: PropTypes.func,
    onTargetSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    onUpdate: PropTypes.func,
    query: queryInterface,
    queryIsRunning: PropTypes.bool,
    queryType: PropTypes.string,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
    queryText: PropTypes.string,
  };

  static defaultProps = {
    queryType: 'query',
    targetsCount: 0,
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
      onStopQuery,
      onUpdate,
      query,
      queryIsRunning,
      queryText,
      queryType,
    } = this.props;

    return (
      <QueryForm
        onCancel={onFormCancel}
        onRunQuery={onRunQuery}
        onSave={onSave}
        onStopQuery={onStopQuery}
        onUpdate={onUpdate}
        query={query}
        queryIsRunning={queryIsRunning}
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
      targetsCount,
    } = this.props;

    if (queryType === 'label') {
      return false;
    }


    return (
      <div>
        <p className={`${baseClass}__target-label`}>
          <span className={`${baseClass}__select-targets`}>Select Targets</span>
          <span className={`${baseClass}__targets-count`}> {targetsCount} unique {targetsCount === 1 ? 'host' : 'hosts' }</span>
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
    const { onTextEditorInputChange, queryIsRunning, queryText, queryType } = this.props;
    const { onLoad, renderForm, renderTargetsInput } = this;

    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>{queryType === 'label' ? 'New Label Query' : 'New Query'}</h1>
        <div className={`${baseClass}__text-editor-wrapper`}>
          <AceEditor
            enableBasicAutocompletion
            enableLiveAutocompletion
            editorProps={{ $blockScrolling: Infinity }}
            mode="kolide"
            minLines={2}
            maxLines={20}
            name="query-editor"
            onLoad={onLoad}
            onChange={onTextEditorInputChange}
            readOnly={queryIsRunning}
            setOptions={{ enableLinking: true }}
            showGutter
            showPrintMargin={false}
            theme="kolide"
            value={queryText}
            width="100%"
            fontSize={14}
          />
        </div>
        {renderTargetsInput()}
        {renderForm()}
      </div>
    );
  }
}

export default QueryComposer;
