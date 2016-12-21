import React, { Component, PropTypes } from 'react';
import { filter } from 'lodash';

import Icon from 'components/Icon';
import InputField from 'components/forms/fields/InputField';
import labelInterface from 'interfaces/label';
import PanelGroup from 'components/side_panels/HostSidePanel/PanelGroup';
import SecondarySidePanelContainer from 'components/side_panels/SecondarySidePanelContainer';

const baseClass = 'host-side-panel';

class HostSidePanel extends Component {
  static propTypes = {
    labels: PropTypes.arrayOf(labelInterface),
    onAddLabelClick: PropTypes.func,
    onLabelClick: PropTypes.func,
    selectedLabel: labelInterface,
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
    const { labels, onAddLabelClick, onLabelClick, selectedLabel } = this.props;
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
        />
        <hr className={`${baseClass}__hr`} />
        <PanelGroup
          groupItems={hostStatusLabels}
          onLabelClick={onLabelClick}
          selectedLabel={selectedLabel}
        />
        <hr className={`${baseClass}__hr`} />
        <PanelGroup
          groupItems={hostPlatformLabels}
          onLabelClick={onLabelClick}
          selectedLabel={selectedLabel}
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
        />
        <hr className={`${baseClass}__hr`} />
        <button className={`${baseClass}__add-label-btn button button--unstyled`} onClick={onAddLabelClick}>
          ADD NEW LABEL
          <Icon name="label" className={`${baseClass}__add-label-btn--icon ${baseClass}__add-label-btn--icon-label`} />
        </button>
      </SecondarySidePanelContainer>
    );
  }
}

export default HostSidePanel;
