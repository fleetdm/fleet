import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import Button from '../../buttons/Button';
import {
  availability,
  columnsToRender,
  displayTypeForDataType,
  numAdditionalColumns,
  shouldShowAllColumns,
} from './helpers';
import componentStyles from './styles';
import { osqueryTables } from '../../../utilities/osquery_tables';


class QuerySidePanel extends Component {
  static propTypes = {
    onOsqueryTableSelect: PropTypes.func,
    onTextEditorInputChange: PropTypes.func,
    selectedOsqueryTable: PropTypes.object,
  };

  componentWillMount () {
    const { selectedOsqueryTable } = this.props;
    const showAllColumns = shouldShowAllColumns(selectedOsqueryTable);

    this.setState({ showAllColumns });
  }

  componentWillReceiveProps (nextProps) {
    const { selectedOsqueryTable } = nextProps;

    if (this.props.selectedOsqueryTable !== selectedOsqueryTable) {
      const showAllColumns = shouldShowAllColumns(selectedOsqueryTable);

      this.setState({ showAllColumns });
    }

    return false;
  }

  onSelectTable = ({ target }) => {
    const { onOsqueryTableSelect } = this.props;
    const { value: tableName } = target;

    onOsqueryTableSelect(tableName);

    return false;
  }

  onShowAllColumns = () => {
    this.setState({ showAllColumns: true });
  }

  onSuggestedQueryClick = (query) => {
    return (evt) => {
      evt.preventDefault();

      const { onTextEditorInputChange } = this.props;

      return onTextEditorInputChange(query);
    };
  };

  renderColumns = () => {
    const { selectedOsqueryTable } = this.props;
    const { showAllColumns } = this.state;
    const { columnNameStyles, columnWrapperStyles, helpStyles } = componentStyles;
    const columns = columnsToRender(selectedOsqueryTable, showAllColumns);

    return columns.map(column => {
      return (
        <div style={columnWrapperStyles} key={column.name}>
          <span style={columnNameStyles}>{column.name}</span>
          <div>
            <span>{displayTypeForDataType(column.type)}</span>
            <i className="kolidecon-help" style={helpStyles} title={column.description} />
          </div>
        </div>
      );
    });
  }

  renderMoreColumns = () => {
    const { columnWrapperStyles, numMoreColumnsStyles, showAllColumnsStyles } = componentStyles;
    const { selectedOsqueryTable } = this.props;
    const { showAllColumns } = this.state;
    const { onShowAllColumns } = this;

    if (showAllColumns) {
      return false;
    }

    return (
      <div style={[columnWrapperStyles, { display: 'flex', justifyContent: 'space-between' }]}>
        <span style={numMoreColumnsStyles}>{numAdditionalColumns(selectedOsqueryTable)} MORE COLUMNS</span>
        <span onClick={onShowAllColumns} style={showAllColumnsStyles}>SHOW</span>
      </div>
    );
  }

  renderSuggestedQueries = () => {
    const { columnWrapperStyles, loadSuggestedQueryStyles, suggestedQueryStyles } = componentStyles;
    const { onSuggestedQueryClick } = this;
    const { selectedOsqueryTable } = this.props;

    return selectedOsqueryTable.examples.map(example => {
      return (
        <div key={example} style={columnWrapperStyles}>
          <span style={suggestedQueryStyles}>{example}</span>
          <Button
            onClick={onSuggestedQueryClick(example)}
            style={loadSuggestedQueryStyles}
            text="LOAD"
          />
        </div>
      );
    });
  }

  renderTableSelect = () => {
    const { onSelectTable } = this;
    const { selectedOsqueryTable } = this.props;

    return (
      <div className="kolide-dropdown-wrapper">
        <select className="kolide-dropdown" onChange={onSelectTable} value={selectedOsqueryTable.name}>
          {osqueryTables.map(table => {
            return <option key={table.name} value={table.name}>{table.name}</option>;
          })}
        </select>
      </div>
    );
  }

  render () {
    const {
      containerStyles,
      platformsTextStyles,
      sectionHeader,
      tableDescriptionStyles,
    } = componentStyles;
    const {
      renderColumns,
      renderMoreColumns,
      renderTableSelect,
      renderSuggestedQueries,
    } = this;
    const { selectedOsqueryTable: { description, platform } } = this.props;

    return (
      <div style={containerStyles}>
        <p style={sectionHeader}>Choose a Table</p>
        {renderTableSelect()}
        <p style={tableDescriptionStyles}>{description}</p>
        <div>
          <p style={sectionHeader}>OS Availability</p>
          <p style={platformsTextStyles}>{availability(platform)}</p>
        </div>
        <div>
          <p style={sectionHeader}>Columns</p>
          {renderColumns()}
          {renderMoreColumns()}
        </div>
        <div>
          <p style={sectionHeader}>Joins</p>
        </div>
        <div>
          <p style={sectionHeader}>Suggested Queries</p>
          {renderSuggestedQueries()}
        </div>
      </div>
    );
  }
}

export default radium(QuerySidePanel);
