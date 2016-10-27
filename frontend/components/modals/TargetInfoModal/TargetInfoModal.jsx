import React, { Component, PropTypes } from 'react';
import AceEditor from 'react-ace';
import classnames from 'classnames';

import Button from '../../buttons/Button';
import { headerClassName } from './helpers';
import hostHelpers from '../../hosts/HostDetails/helpers';
import Modal from '../Modal';
import ShadowBox from '../../ShadowBox';
import ShadowBoxInput from '../../forms/fields/ShadowBoxInput';
import targetInterface from '../../../interfaces/target';

const baseClass = 'target-info-modal';

class TargetInfoModal extends Component {
  static propTypes = {
    className: PropTypes.string,
    onAdd: PropTypes.func,
    onExit: PropTypes.func,
    target: targetInterface.isRequired,
  };

  renderButtons = () => {
    const { onAdd, onExit } = this.props;

    return (
      <div className={`${baseClass}__btn-wrapper`}>
        <Button className={`${baseClass}__cancel-btn`} text="CANCEL" variant="inverse" onClick={onExit} />
        <Button className={`${baseClass}__add-btn`} text="ADD TO TARGETS" onClick={onAdd} />
      </div>
    );
  }

  renderHeader = () => {
    const { target } = this.props;
    const { label } = target;
    const className = headerClassName(target);

    return (
      <span className={`${baseClass}__header`}>
        <i className={className} />
        <span>{label}</span>
      </span>
    );
  }

  renderHostModal = () => {
    const { className, onExit, target } = this.props;
    const hostBaseClass = `${baseClass}__host`;
    const {
      ip,
      mac,
      memory,
      platform,
      os_version: osVersion,
      osquery_version: osqueryVersion,
      status,
    } = target;
    const isOnline = status === 'online';
    const isOffline = status === 'offline';
    const { renderButtons, renderHeader } = this;
    const statusClassName = classnames(`${hostBaseClass}__status`, {
      '--is-online': isOnline,
      '--is-offline': isOffline,
    });

    return (
      <Modal
        className={`${className} host-modal`}
        onExit={onExit}
        title={renderHeader()}
      >
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
        {renderButtons()}
      </Modal>
    );
  }

  renderLabelModal = () => {
    const { className, onExit, target } = this.props;
    const {
      description,
      hosts = [],
      query,
    } = target;
    const labelBaseClass = `${baseClass}__label`;
    const { renderButtons, renderHeader } = this;

    return (
      <Modal
        className={`${className} label-modal`}
        onExit={onExit}
        title={renderHeader()}
      >
        <p className={`${labelBaseClass}__description`}>{description}</p>
        <div className={`${labelBaseClass}__text-editor-wrapper`}>
          <AceEditor
            editorProps={{ $blockScrolling: Infinity }}
            mode="kolide"
            minLines={4}
            maxLines={4}
            name="modal-label-query"
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
        {renderButtons()}
      </Modal>
    );
  }

  render () {
    const { renderHostModal, renderLabelModal } = this;
    const { target_type: targetType } = this.props.target;

    if (targetType === 'hosts') {
      return renderHostModal();
    }

    return renderLabelModal();
  }
}

export default TargetInfoModal;
