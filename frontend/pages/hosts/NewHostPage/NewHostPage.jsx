import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { map, noop } from 'lodash';
import classnames from 'classnames';

import { renderFlash } from 'redux/nodes/notifications/actions';
import Icon from 'components/Icon';
import { copyText } from './helpers';

const HOST_TABS = {
  FIRST: 'What Does This Script Do?',
  SECOND: 'Additional Script Options',
};

const baseClass = 'new-host';

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
    const { selectedTab } = this.state;
    const { onSetActiveTab } = this;

    return map(HOST_TABS, (tab) => {
      const selected = selectedTab === tab;
      const hostTabHeaderClass = classnames(
        `${baseClass}__tab-header`,
        { [`${baseClass}__tab-header--selected`]: selected }
      );

      return <button className={`button button--unstyled ${hostTabHeaderClass}`} onClick={onSetActiveTab(tab)} key={tab}>{tab}</button>;
    });
  }

  render () {
    const { method1Text, method1TextCopied, method2Text, method2TextCopied } = this.state;
    const { onCopyText, renderHostTabContent, renderHostTabHeaders } = this;

    const method1IconClasses = classnames(
      `${baseClass}__clipboard-icon`,
      {
        [`${baseClass}__clipboard-icon--copied`]: method1TextCopied,
      }
    );
    const method2IconClasses = classnames(
      `${baseClass}__clipboard-icon`,
      {
        [`${baseClass}__clipboard-icon--copied`]: method2TextCopied,
      }
    );

    return (
      <div className={baseClass}>
        <div className={`${baseClass}__section-wrap body-wrap`}>
          <p className={`${baseClass}__title`}>Method 1 - One Liner</p>
          <div className={`${baseClass}__input-wrap`}>
            <input id="method1" className={`${baseClass}__input`} value={method1Text} readOnly />
            {method1TextCopied && <span className={`${baseClass}__clipboard-text`}>copied!</span>}
            <a href="#copyMethod1" onClick={onCopyText(method1Text, '#method1')}><Icon name="clipboard" className={method1IconClasses} /></a>
          </div>
          <div className={`${baseClass}__tab-wrap`}>
            {renderHostTabHeaders()}
            <div className={`${baseClass}__tab-body`}>
              {renderHostTabContent()}
            </div>
          </div>
        </div>
        <div className={`${baseClass}__section-wrap body-wrap`}>
          <p className={`${baseClass}__title ${baseClass}__title--wide`}>Method 2 - Your osqueryd with Kolide config</p>
          <div className={`${baseClass}__input-wrap`}>
            <input id="method2" className={`${baseClass}__input`} value={method2Text} readOnly />
            {method2TextCopied && <span className={`${baseClass}__clipboard-text`}>copied!</span>}
            <a href="#copyMethod2" onClick={onCopyText(method2Text, '#method2')}><Icon name="clipboard" className={method2IconClasses} /></a>
          </div>
          <p className={`${baseClass}__text`}>This method allows you to configure an existing osqueryd installation to work with Kolide. The <code>--config_endpoints</code> flag allows us to point your osqueryd installation to your Kolide configuration.</p>
        </div>
        <div className={`${baseClass}__section-wrap body-wrap`}>
          <p className={`${baseClass}__title`}>Method 3 - Need More Methods?</p>
          <p className={`${baseClass}__text`}>Many IT automation frameworks offer direct recipes and scripts for deploying osquery. Choose a method below to learn more.</p>
        </div>
      </div>
    );
  }
}

export default connect()(NewHostPage);
