import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import classnames from 'classnames';
import 'brace/mode/sql';
import 'brace/ext/linking';

import './mode';
import './theme';

const baseClass = 'kolide-ace';

class KolideAce extends Component {
  static propTypes = {
    error: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    onLoad: PropTypes.func,
    value: PropTypes.string,
    readOnly: PropTypes.bool,
    wrapperClassName: PropTypes.string,
  };

  static defaultProps = {
    name: 'query-editor',
  };

  render () {
    const {
      error,
      name,
      onChange,
      onLoad,
      readOnly,
      value,
      wrapperClassName,
    } = this.props;

    const wrapperClass = classnames(wrapperClassName, {
      [`${baseClass}__wrapper--error`]: error,
    });

    return (
      <div className={wrapperClass}>
        <div className={`${baseClass}__error-field`}>{error}</div>
        <AceEditor
          enableBasicAutocompletion
          enableLiveAutocompletion
          editorProps={{ $blockScrolling: Infinity }}
          mode="kolide"
          minLines={2}
          maxLines={20}
          name={name}
          onChange={onChange}
          onLoad={onLoad}
          readOnly={readOnly}
          setOptions={{ enableLinking: true }}
          showGutter
          showPrintMargin={false}
          theme="kolide"
          value={value}
          width="100%"
          fontSize={14}
        />
      </div>
    );
  }
}

export default KolideAce;
