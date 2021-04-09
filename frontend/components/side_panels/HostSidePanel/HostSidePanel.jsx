import React, { Component } from "react";
import PropTypes from "prop-types";
import { filter } from "lodash";

import KolideIcon from "components/icons/KolideIcon";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import labelInterface from "interfaces/label";
import PanelGroup from "components/side_panels/HostSidePanel/PanelGroup";
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";
import statusLabelsInterface from "interfaces/status_labels";

const baseClass = "host-side-panel";

class HostSidePanel extends Component {
  static propTypes = {
    labels: PropTypes.arrayOf(labelInterface),
    onAddLabelClick: PropTypes.func,
    onLabelClick: PropTypes.func,
    selectedFilter: PropTypes.string,
    statusLabels: statusLabelsInterface,
  };

  constructor(props) {
    super(props);

    this.state = { labelFilter: "" };
  }

  onFilterLabels = (labelFilter) => {
    const lowerLabelFilter = labelFilter.toLowerCase();
    this.setState({ labelFilter: lowerLabelFilter });

    return false;
  };

  render() {
    const {
      labels,
      onAddLabelClick,
      onLabelClick,
      selectedFilter,
      statusLabels,
    } = this.props;
    const { labelFilter } = this.state;
    const { onFilterLabels } = this;
    const allHostLabels = filter(labels, { type: "all" });
    const hostStatusLabels = filter(labels, { type: "status" });
    const hostPlatformLabels = filter(labels, (label) => {
      return label.type === "platform" && label.count > 0;
    });
    const customLabels = filter(labels, (label) => {
      const lowerDisplayText = label.display_text.toLowerCase();

      return label.type === "custom" && lowerDisplayText.match(labelFilter);
    });

    return (
      <SecondarySidePanelContainer className={`${baseClass}`}>
        <h3>Status</h3>
        <PanelGroup
          groupItems={allHostLabels}
          onLabelClick={onLabelClick}
          selectedFilter={selectedFilter}
          type="all-hosts"
        />

        <PanelGroup
          groupItems={hostStatusLabels}
          onLabelClick={onLabelClick}
          statusLabels={statusLabels}
          selectedFilter={selectedFilter}
          type="status"
        />

        <h3>Operating Systems</h3>
        <PanelGroup
          groupItems={hostPlatformLabels}
          onLabelClick={onLabelClick}
          selectedFilter={selectedFilter}
          type="platform"
        />

        <h3 className="title">Labels</h3>
        <div
          className={`${baseClass}__panel-group-item ${baseClass}__panel-group-item--filter`}
        >
          <InputField
            name="tags-filter"
            onChange={onFilterLabels}
            placeholder="Filter labels by name..."
            value={labelFilter}
            inputWrapperClass={`${baseClass}__filter-labels`}
          />
          <KolideIcon name="search" />
        </div>
        <PanelGroup
          groupItems={customLabels}
          onLabelClick={onLabelClick}
          selectedFilter={selectedFilter}
          type="label"
        />

        <Button
          variant="grey"
          onClick={onAddLabelClick}
          className={`${baseClass}__add-label-btn`}
        >
          Add new label
        </Button>
      </SecondarySidePanelContainer>
    );
  }
}

export default HostSidePanel;
