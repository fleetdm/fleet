import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import classnames from 'classnames';

import { renderFlash } from 'redux/nodes/notifications/actions';
import Icon from 'components/Icon';
import { copyText } from './helpers';
import AnsibleImage from '../../../../assets/images/Ansible.png';
import ChefImage from '../../../../assets/images/Chef.png';
import PuppetImage from '../../../../assets/images/Puppet.png';

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
      selectedTab: HOST_TABS.FIRST,
    };
  }

  onCopyText = (text, elementId) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { method1Text } = this.state;

      if (copyText(elementId)) {
        dispatch(renderFlash('success', 'Text copied to clipboard'));
      } else {
        dispatch(renderFlash('error', 'Text not copied. Use CMD + C to copy text'));
      }

      if (text === method1Text) {
        this.setState({
          method1TextCopied: true,
        });
      }

      setTimeout(() => {
        this.setState({
          method1TextCopied: false,
        });

        return false;
      }, 1500);

      return false;
    };
  }

  render () {
    const { method1Text, method1TextCopied } = this.state;
    const { onCopyText } = this;

    const method1IconClasses = classnames(
      `${baseClass}__clipboard-icon`,
      {
        [`${baseClass}__clipboard-icon--copied`]: method1TextCopied,
      }
    );

    return (
      <div className={baseClass}>
        <section className={`${baseClass}__section-wrap body-wrap`}>
          <h1 className={`${baseClass}__title`}>Kolide Installation Instructions</h1>
          <div className={`${baseClass}__input-wrap`}>
            <input id="method1" className={`${baseClass}__input`} value={method1Text} readOnly />
            {method1TextCopied && <span className={`${baseClass}__clipboard-text`}>copied!</span>}
            <a href="#copyMethod1" onClick={onCopyText(method1Text, '#method1')}><Icon name="clipboard" className={method1IconClasses} /></a>
          </div>

          <div className={`${baseClass}__text`}>
            <p>This script does the following:</p>
            <ol className="kolide-ol">
              <li>Detects operating system.</li>
              <li>Checks for any existing osqueryd installation.</li>
              <li>Installs osqueryd and ships your config to communicate with Kolide.</li>
            </ol>
          </div>

          <p className={`${baseClass}__view-script`}><button className="button button--unstyled">View The Script</button></p>
        </section>

        <section className={`${baseClass}__section-wrap body-wrap`}>
          <h1 className={`${baseClass}__title`}>Need More Methods?</h1>

          <p className={`${baseClass}__text`}>Many IT automation frameworks offer direct recipes and scripts for deploying osquery. <br />Choose a method below to learn more.</p>

          <ul className={`${baseClass}__more-methods`}>
            <li>
              <button className="button button--unstyled" title="Chef">
                <img src={ChefImage} alt="Chef" />
              </button>
            </li>
            <li>
              <button className="button button--unstyled" title="Ansible">
                <img src={AnsibleImage} alt="Ansible" />
              </button>
            </li>
            <li>
              <button className="button button--unstyled" title="Puppet">
                <img src={PuppetImage} alt="Puppet" />
              </button>
            </li>
          </ul>
        </section>
      </div>
    );
  }
}

export default connect()(NewHostPage);
