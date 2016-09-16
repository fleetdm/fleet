import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import { Link } from 'react-router';
import componentStyles from './styles';

class StackedWhiteBoxes extends Component {
  static propTypes = {
    children: PropTypes.element,
    headerText: PropTypes.string,
    leadText: PropTypes.string,
    previousLocation: PropTypes.string,
    style: PropTypes.object,
  };

  static defaultProps = {
    style: {},
  };

  renderBackButton = () => {
    const { previousLocation } = this.props;
    const { exStyles, exWrapperStyles } = componentStyles;

    if (!previousLocation) return false;

    return (
      <div style={exWrapperStyles}>
        <Link style={exStyles} to={previousLocation}>x</Link>
      </div>
    );
  }

  renderHeader = () => {
    const { headerStyles, headerWrapperStyles } = componentStyles;
    const { headerText, style } = this.props;

    return (
      <div style={[headerWrapperStyles, style.headerWrapper]}>
        <p style={headerStyles}>{headerText}</p>
      </div>
    );
  }

  render () {
    const { children, leadText } = this.props;
    const {
      boxStyles,
      containerStyles,
      smallTabStyles,
      tabStyles,
      textStyles,
    } = componentStyles;
    const { renderBackButton, renderHeader } = this;

    return (
      <div style={containerStyles}>
        <div style={smallTabStyles} />
        <div style={tabStyles} />
        <div style={boxStyles}>
          {renderBackButton()}
          {renderHeader()}
          <p style={textStyles}>{leadText}</p>
          {children}
        </div>
      </div>
    );
  }
}

export default radium(StackedWhiteBoxes);
