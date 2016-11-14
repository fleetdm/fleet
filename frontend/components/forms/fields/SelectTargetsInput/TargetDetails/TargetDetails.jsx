import React, { Component } from 'react';
import AceEditor from 'react-ace';
import classnames from 'classnames';

import hostHelpers from 'components/hosts/HostDetails/helpers';
import ShadowBox from 'components/ShadowBox';
import ShadowBoxInput from 'components/forms/fields/ShadowBoxInput';
import targetInterface from 'interfaces/target';

const baseClass = 'target-details';

class TargetDetails extends Component {
  static propTypes = {
    target: targetInterface,
  };

  renderHost = () => {
    const {
      ip,
      mac,
      memory,
      osqueryVersion,
      osVersion,
      platform,
      status,
    } = this.props.target;
    const hostBaseClass = 'host-target';
    const isOnline = status === 'online';
    const isOffline = status === 'offline';
    const statusClassName = classnames(
      `${hostBaseClass}__status`,
      { [`${hostBaseClass}__status--is-online`]: isOnline },
      { [`${hostBaseClass}__status--is-offline`]: isOffline },
    );

    return (
      <div>
        <p className={statusClassName}>{status}</p>
        <ShadowBox>
          <table className={`${baseClass}__table`}>
            <tbody>
              <tr>
                <th>IP Address</th>
                <td>{ip}</td>
              </tr>
              <tr>
                <th>MAC Address</th>
                <td>{mac}</td>
              </tr>
              <tr>
                <th>Platform</th>
                <td>
                  <i className={hostHelpers.platformIconClass(platform)} />
                  <span className={`${hostBaseClass}__platform-text`}>{platform}</span>
                </td>
              </tr>
              <tr>
                <th>Operating System</th>
                <td>{osVersion}</td>
              </tr>
              <tr>
                <th>Osquery Version</th>
                <td>{osqueryVersion}</td>
              </tr>
              <tr>
                <th>Memory</th>
                <td>{hostHelpers.humanMemory(memory)}</td>
              </tr>
            </tbody>
          </table>
        </ShadowBox>
        <div className={`${hostBaseClass}__labels-wrapper`}>
          <div className={`${hostBaseClass}__labels-wrapper--header`}>
            <i className="kolidecon-label" />
            <span>Labels</span>
          </div>
        </div>
      </div>
    );
  }

  renderLabel = () => {
    const { hosts, query } = this.props.target;
    const labelBaseClass = 'label-target';

    return (
      <div>
        <div className={`${labelBaseClass}__text-editor-wrapper`}>
          <AceEditor
            editorProps={{ $blockScrolling: Infinity }}
            mode="kolide"
            minLines={4}
            maxLines={4}
            name="label-query"
            readOnly
            setOptions={{ wrap: true }}
            showGutter={false}
            showPrintMargin={false}
            theme="kolide"
            value={query}
            width="100%"
          />
        </div>
        <div className={`${labelBaseClass}__search-section`}>
          <ShadowBoxInput
            iconClass="kolidecon-search"
            name="search-hosts"
            placeholder="SEARCH HOSTS"
          />
          <div className={`${labelBaseClass}__num-hosts-section`}>
            <span className="num-hosts">{hosts.length} HOSTS</span>
          </div>
        </div>
        <ShadowBox>
          <table className={`${baseClass}__table`}>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>Status</th>
                <th>Platform</th>
                <th>Location</th>
                <th>MAC</th>
              </tr>
            </thead>
            <tbody>
              {hosts.map((host) => {
                return (
                  <tr className="__label-row" key={`host-${host.id}`}>
                    <td>{host.hostname}</td>
                    <td>{host.status}</td>
                    <td><i className={hostHelpers.platformIconClass(host.platform)} /></td>
                    <td>{host.ip}</td>
                    <td>{host.mac}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </ShadowBox>
      </div>
    );
  }

  render () {
    const { target } = this.props;

    if (!target) {
      return false;
    }

    const { target_type: targetType } = target;
    const { renderHost, renderLabel } = this;

    if (targetType === 'labels') {
      return renderLabel();
    }

    return renderHost();
  }
}

export default TargetDetails;
