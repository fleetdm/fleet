import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { noop } from 'lodash';

import { getOsqueryOptions } from 'redux/nodes/osquery/actions'

import AceEditor from "react-ace";

const baseClass = 'osquery-options';

class OsqueryOptionsPage extends Component {
  static propTypes = {
    osqueryOptions: PropTypes.object, // eslint-disable-line react/forbid-prop-types
    dispatch: PropTypes.func,
  };

  static defaultProps = {
    dispatch: noop,
  }

  componentDidMount() {
    const { dispatch } = this.props;

    dispatch(getOsqueryOptions())
      .catch(() => false);
  }

  render () {
    const { osqueryOptions } = this.props;
    return (
      <div className={`${baseClass} body-wrap`}>
        <h1>Osquery Options</h1>
        {/* {osqueryOptionsRender} */}
        <AceEditor
          mode="json"
          theme="kolide"
          width="60%"
          minLines={2}
          maxLines={50}
          editorProps={{ $blockScrolling: Infinity }}
          value={JSON.stringify(osqueryOptions, null, "\t")}
          tabSize={2}
        />
      </div>
    );
  };
};

const mapStateToProps = (state) => {
    const { osquery } = state;
    const { options } = osquery;
    console.log(options)
    return {
        osqueryOptions: options,
    };
}

export default connect(mapStateToProps)(OsqueryOptionsPage);