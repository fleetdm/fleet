import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import classnames from 'classnames';
import 'brace/mode/sql';
import 'brace/ext/linking';
import 'brace/ext/language_tools';

import './mode';
import './theme';

const baseClass = 'kolide-ace';

class KolideAce extends Component {
  static propTypes = {
    error: PropTypes.string,
    fontSize: PropTypes.number,
    label: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    onLoad: PropTypes.func,
    value: PropTypes.string,
    readOnly: PropTypes.bool,
    showGutter: PropTypes.bool,
    wrapEnabled: PropTypes.bool,
    wrapperClassName: PropTypes.string,
  };

  static defaultProps = {
    fontSize: 14,
    name: 'query-editor',
    showGutter: true,
    wrapEnabled: false,
  };

  renderLabel = () => {
    const { error, label } = this.props;

    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: error,
    });

    return (
      <p className={labelClassName}>{error || label}</p>
    );
  }

  render () {
    const {
      error,
      fontSize,
      name,
      onChange,
      onLoad,
      readOnly,
      showGutter,
      value,
      wrapEnabled,
      wrapperClassName,
    } = this.props;
    const { renderLabel } = this;

    const wrapperClass = classnames(wrapperClassName, {
      [`${baseClass}__wrapper--error`]: error,
    });

    return (
      <div className={wrapperClass}>
        {renderLabel()}
        <AceEditor
          enableBasicAutocompletion
          enableLiveAutocompletion
          editorProps={{ $blockScrolling: Infinity }}
          fontSize={fontSize}
          mode="kolide"
          minLines={2}
          maxLines={20}
          name={name}
          onChange={onChange}
          onLoad={onLoad}
          readOnly={readOnly}
          setOptions={{ enableLinking: true }}
          showGutter={showGutter}
          showPrintMargin={false}
          theme="kolide"
          value={value}
          width="100%"
          wrapEnabled={wrapEnabled}
        />
      </div>
    );
  }
}

export default KolideAce;
