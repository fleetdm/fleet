import React, { Component, PropTypes } from 'react';
import { Style } from 'radium';
import Footer from './Footer';
import componentStyles from './styles';
import globalStyles from '../../styles/global';

export class App extends Component {
  static propTypes = {
    children: PropTypes.element,
  };

  render () {
    const { children } = this.props;
    const { containerStyles, childWrapperStyles } = componentStyles;

    return (
      <div style={containerStyles}>
        <Style rules={globalStyles} />
        <div style={childWrapperStyles}>
          {children}
        </div>
        <Footer />
      </div>
    );
  }
}

export default App;
