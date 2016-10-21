import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { map, noop } from 'lodash';

import componentStyles from './styles';
import { copyText } from './helpers';
import Icon from '../../../components/icons/Icon';
import { renderFlash } from '../../../redux/nodes/notifications/actions';

const HOST_TABS = {
  FIRST: 'What Does This Script Do?',
  SECOND: 'Additional Script Options',
};

export class NewHostPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = {
      method1Text: 'curl https://kolide.acme.com/install/osquery.sh | sudo sh',
      method1TextCopied: false,
      method2Text: 'osqueryd --config_endpoint="https://kolide.acme.com/api/v1/osquery/autoconfigure"',
      method2TextCopied: false,
      selectedTab: HOST_TABS.FIRST,
    };
  }

  onCopyText = (text, elementId) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { method1Text, method2Text } = this.state;

      if (copyText(elementId)) {
        dispatch(renderFlash('success', 'Text copied to clipboard'));
      } else {
        dispatch(renderFlash('error', 'Text not copied. Use CMD + C to copy text'));
      }

      if (text === method1Text) {
        this.setState({
          method1TextCopied: true,
          method2TextCopied: false,
        });
      }

      if (text === method2Text) {
        this.setState({
          method1TextCopied: false,
          method2TextCopied: true,
        });
      }

      setTimeout(() => {
        this.setState({
          method1TextCopied: false,
          method2TextCopied: false,
        });

        return false;
      }, 1500);

      return false;
    };
  }

  onSetActiveTab = (selectedTab) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({ selectedTab });

      return false;
    };
  }

  renderHostTabContent = () => {
    const { selectedTab } = this.state;

    if (selectedTab === HOST_TABS.FIRST) {
      return (
        <div>
          <p style={{ marginTop: 0 }}>This script does the following:</p>
          <ol className="kolide-ol">
            <li>Detects operating system.</li>
            <li>Checks for any existing osqueryd installation.</li>
            <li>Installs osqueryd and ships your config to communicate with Kolide.</li>
          </ol>
        </div>
      );
    }

    return false;
  }

  renderHostTabHeaders = () => {
    const { hostTabHeaderStyles } = componentStyles;
    const { selectedTab } = this.state;
    const { onSetActiveTab } = this;

    return map(HOST_TABS, (tab) => {
      const selected = selectedTab === tab;

      return <button className="btn--unstyled" onClick={onSetActiveTab(tab)} key={tab} style={hostTabHeaderStyles(selected)}>{tab}</button>;
    });
  }

  render () {
    const {
      clipboardIconStyles,
      clipboardTextStyles,
      headerStyles,
      inputStyles,
      textStyles,
      scriptInfoWrapperStyles,
      selectedTabContentStyles,
      sectionWrapperStyles,
    } = componentStyles;
    const { method1Text, method1TextCopied, method2Text, method2TextCopied } = this.state;
    const { onCopyText, renderHostTabContent, renderHostTabHeaders } = this;
    const method2HeaderStyles = { ...headerStyles, width: '626px' };

    return (
      <div>
        <div style={sectionWrapperStyles}>
          <p style={headerStyles}>Method 1 - One Liner</p>
          <div style={{ position: 'relative' }}>
            <input id="method1" style={inputStyles} value={method1Text} readOnly />
            {method1TextCopied && <span style={clipboardTextStyles}>copied!</span>}
            <Icon name="clipboard" onClick={onCopyText(method1Text, '#method1')} style={clipboardIconStyles} variant={method1TextCopied ? 'copied' : 'default'} />
          </div>
          <div style={scriptInfoWrapperStyles}>
            {renderHostTabHeaders()}
            <div style={selectedTabContentStyles}>
              {renderHostTabContent()}
            </div>
          </div>
        </div>
        <div style={sectionWrapperStyles}>
          <p style={method2HeaderStyles}>Method 2 - Your osqueryd with Kolide config</p>
          <div style={{ position: 'relative' }}>
            <input id="method2" style={inputStyles} value={method2Text} readOnly />
            {method2TextCopied && <span style={clipboardTextStyles}>copied!</span>}
            <Icon name="clipboard" onClick={onCopyText(method2Text, '#method2')} style={clipboardIconStyles} variant={method2TextCopied ? 'copied' : 'default'} />
          </div>
          <p style={textStyles}>This method allows you to configure an existing osqueryd installation to work with Kolide. The <span style={{ color: '#AE6DDf', fontFamily: 'SourceCodePro, Oxygen' }}>--config_endpoints</span> flag allows us to point your osqueryd installation to your Kolide configuration.</p>
        </div>
        <div style={sectionWrapperStyles}>
          <p style={headerStyles}>Method 3 - Need More Methods?</p>
          <p style={textStyles}>Many IT automation frameworks offer direct recipes and scripts for deploying osquery. Choose a method below to learn more.</p>
        </div>
      </div>
    );
  }
}

export default connect()(NewHostPage);
