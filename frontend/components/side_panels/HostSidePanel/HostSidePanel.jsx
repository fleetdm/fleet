import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { filter } from 'lodash';

import Icon from 'components/icons/Icon';
import Button from 'components/buttons/Button';
import InputField from 'components/forms/fields/InputField';
import labelInterface from 'interfaces/label';
import PanelGroup from 'components/side_panels/HostSidePanel/PanelGroup';
import SecondarySidePanelContainer from 'components/side_panels/SecondarySidePanelContainer';
import statusLabelsInterface from 'interfaces/status_labels';

const baseClass = 'host-side-panel';

class HostSidePanel extends Component {
  static propTypes = {
    labels: PropTypes.arrayOf(labelInterface),
    onAddLabelClick: PropTypes.func,
    onAddHostClick: PropTypes.func,
    onLabelClick: PropTypes.func,
    selectedLabel: labelInterface,
    statusLabels: statusLabelsInterface,
  };

  constructor (props) {
    super(props);

    this.state = { labelFilter: '' };
  }

  onFilterLabels = (labelFilter) => {
    const lowerLabelFilter = labelFilter.toLowerCase();
    this.setState({ labelFilter: lowerLabelFilter });

    return false;
  }

  render () {
    const { labels, onAddHostClick, onAddLabelClick, onLabelClick, selectedLabel, statusLabels } = this.props;
    const { labelFilter } = this.state;
    const { onFilterLabels } = this;
    const allHostLabels = filter(labels, { type: 'all' });
    const hostStatusLabels = filter(labels, { type: 'status' });
    const hostPlatformLabels = filter(labels, { type: 'platform' });
    const customLabels = filter(labels, (label) => {
      const lowerDisplayText = label.display_text.toLowerCase();

      return label.type === 'custom' &&
        lowerDisplayText.match(labelFilter);
    });

    return (
      <SecondarySidePanelContainer className={`${baseClass}__wrapper`}>
        <PanelGroup
          groupItems={allHostLabels}
          onLabelClick={onLabelClick}
          selectedLabel={selectedLabel}
          type="all-hosts"
        />

        <Button variant="unstyled" onClick={onAddHostClick} className={`${baseClass}__add-hosts`}>
          <Icon name="laptop-plus" className={`${baseClass}__add-hosts-icon`} />
          <span>Add New Host</span>
        </Button>

        <hr className={`${baseClass}__hr`} />
        <PanelGroup
          groupItems={hostStatusLabels}
          onLabelClick={onLabelClick}
          statusLabels={statusLabels}
          selectedLabel={selectedLabel}
          type="status"
        />
        <hr className={`${baseClass}__hr`} />
        <PanelGroup
          groupItems={hostPlatformLabels}
          onLabelClick={onLabelClick}
          selectedLabel={selectedLabel}
          type="platform"
        />
        <hr className={`${baseClass}__hr`} />
        <div className={`${baseClass}__panel-group-item`}>
          <Icon name="label" />
          <span className="title">LABELS</span>
        </div>
        <div className={`${baseClass}__panel-group-item ${baseClass}__panel-group-item--filter`}>
          <InputField
            name="tags-filter"
            onChange={onFilterLabels}
            placeholder="Filter Labels by Name..."
            value={labelFilter}
            inputWrapperClass={`${baseClass}__filter-labels`}
          />
          <Icon name="search" />
        </div>
        <PanelGroup
          groupItems={customLabels}
          onLabelClick={onLabelClick}
          selectedLabel={selectedLabel}
          type="label"
        />
        <hr className={`${baseClass}__hr`} />
        <Button variant="unstyled" onClick={onAddLabelClick} className={`${baseClass}__add-label-btn`}>
          ADD NEW LABEL
          <Icon name="label" className={`${baseClass}__add-label-icon`} />
        </Button>
      </SecondarySidePanelContainer>
    );
  }
}

export default HostSidePanel;
