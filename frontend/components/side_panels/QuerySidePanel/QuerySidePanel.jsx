import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import IconToolTip from "components/IconToolTip";
import osqueryTableInterface from "interfaces/osquery_table";
import { osqueryTableNames } from "utilities/osquery_tables";
import Dropdown from "components/forms/fields/Dropdown";
import FleetIcon from "components/icons/FleetIcon";
import SecondarySidePanelContainer from "../SecondarySidePanelContainer";

import displayTypeForDataType from "./helpers";

import AppleIcon from "../../../../assets/images/icon-apple-dark-20x20@2x.png";
import LinuxIcon from "../../../../assets/images/icon-linux-dark-20x20@2x.png";
import WindowsIcon from "../../../../assets/images/icon-windows-dark-20x20@2x.png";

const baseClass = "query-side-panel";

class QuerySidePanel extends Component {
  static propTypes = {
    onOsqueryTableSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    selectedOsqueryTable: osqueryTableInterface,
  };

  onSelectTable = (value) => {
    const { onOsqueryTableSelect } = this.props;

    onOsqueryTableSelect(value);

    return false;
  };

  onSuggestedQueryClick = (query) => {
    return (evt) => {
      evt.preventDefault();

      const { onTextEditorInputChange } = this.props;

      return onTextEditorInputChange(query);
    };
  };

  renderColumns = () => {
    const { selectedOsqueryTable } = this.props;
    const columns = selectedOsqueryTable.columns;
    const columnBaseClass = "query-column-list";

    return columns.map((column) => {
      return (
        <li key={column.name} className={`${columnBaseClass}__item`}>
          <span className={`${columnBaseClass}__name`}>{column.name}</span>
          <IconToolTip text={column.description} />
          <div className={`${columnBaseClass}__description`}>
            <span className={`${columnBaseClass}__type`}>
              {displayTypeForDataType(column.type)}
            </span>
          </div>
        </li>
      );
    });
  };

  renderTableSelect = () => {
    const { onSelectTable } = this;
    const { selectedOsqueryTable } = this.props;

    const tableNames = osqueryTableNames.map((name) => {
      return { label: name, value: name };
    });

    return (
      <Dropdown
        options={tableNames}
        value={selectedOsqueryTable.name}
        onChange={onSelectTable}
        placeholder="Choose Table..."
      />
    );
  };

  render() {
    const { renderColumns, renderTableSelect } = this;
    const {
      selectedOsqueryTable: { description, platforms },
    } = this.props;

    const iconClasses = classnames([`${baseClass}__icon`], "icon");

    return (
      <SecondarySidePanelContainer className={baseClass}>
        <div className={`${baseClass}__choose-table`}>
          <h2 className={`${baseClass}__header`}>Tables</h2>
          {renderTableSelect()}
          <p className={`${baseClass}__description`}>{description}</p>
        </div>

        <div className={`${baseClass}__os-availability`}>
          <h2 className={`${baseClass}__header`}>OS Availability</h2>
          <ul className={`${baseClass}__platforms`}>
            {platforms.map((platform) => {
              if (platform === "all") {
                return (
                  <li key={platform}>
                    <FleetIcon name="hosts" /> {platform}
                  </li>
                );
              } else if (platform === "freebsd") {
                return (
                  <li key={platform}>
                    <FleetIcon name="single-host" /> {platform}
                  </li>
                );
              }
              platform = platform.toLowerCase();
              let icon = (
                <img
                  src={AppleIcon}
                  alt={`${platform} icon`}
                  className={iconClasses}
                />
              );
              if (platform === "linux") {
                icon = (
                  <img
                    src={LinuxIcon}
                    alt={`${platform} icon`}
                    className={iconClasses}
                  />
                );
              } else if (platform === "windows") {
                icon = (
                  <img
                    src={WindowsIcon}
                    alt={`${platform} icon`}
                    className={iconClasses}
                  />
                );
              }

              return (
                <li key={platform}>
                  {icon} {platform}
                </li>
              );
            })}
          </ul>
        </div>

        <div className={`${baseClass}__columns`}>
          <h2 className={`${baseClass}__header`}>Columns</h2>
          <ul className={`${baseClass}__column-list`}>{renderColumns()}</ul>
        </div>
      </SecondarySidePanelContainer>
    );
  }
}

export default QuerySidePanel;
