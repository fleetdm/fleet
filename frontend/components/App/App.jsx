import React, { Component, PropTypes } from 'react';
import { Style } from 'radium';
import globalStyles from '../../styles/global';

export class App extends Component {
  static propTypes = {
    children: PropTypes.element,
  };

  render () {
    const { children } = this.props;

    return (
      <div>
        <Style rules={globalStyles} />
        {children}
      </div>
    );
  }
}

export default App;
