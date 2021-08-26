import React, { Component } from "react";
import PropTypes from "prop-types";
import AceEditor from "react-ace";
import { isEqual, sortBy } from "lodash";

import FleetIcon from "components/icons/FleetIcon";
import queryInterface from "interfaces/query";
import Dropdown from "components/forms/fields/Dropdown";

const baseClass = "search-pack-query";

class SearchPackQuery extends Component {
  static propTypes = {
    allQueries: PropTypes.arrayOf(queryInterface),
    onSelectQuery: PropTypes.func,
    selectedQuery: queryInterface,
  };

  static defaultProps = {
    allQueries: [],
  };

  constructor(props) {
    super(props);

    this.state = {
      queryDropdownOptions: [],
    };
  }

  componentWillMount() {
    const { allQueries } = this.props;

    const queryDropdownOptions = allQueries.map((query) => {
      return { label: query.name, value: String(query.id) };
    });

    this.setState({
      queryDropdownOptions: sortBy(queryDropdownOptions, ["label"]),
    });
  }

  componentWillReceiveProps(nextProps) {
    const { allQueries } = nextProps;

    if (!isEqual(allQueries, this.props.allQueries)) {
      const queryDropdownOptions = allQueries.map((query) => {
        return { label: query.name, value: String(query.id) };
      });

      this.setState({
        queryDropdownOptions: sortBy(queryDropdownOptions, ["label"]),
      });
    }
  }

  renderHeader = () => {
    const { selectedQuery } = this.props;
    if (selectedQuery) {
      return (
        <h1 className={`${baseClass}__title`}>
          <FleetIcon name="query" /> {selectedQuery.name}
        </h1>
      );
    }

    return <h1 className={`${baseClass}__title`}>Choose Query</h1>;
  };

  renderQuery = () => {
    const { selectedQuery } = this.props;
    if (selectedQuery) {
      return (
        <AceEditor
          editorProps={{ $blockScrolling: Infinity }}
          mode="fleet"
          minLines={1}
          maxLines={3}
          name="pack-query"
          readOnly
          setOptions={{ wrap: true }}
          showGutter={false}
          showPrintMargin={false}
          theme="fleet"
          value={selectedQuery.query}
          width="100%"
          fontSize={14}
        />
      );
    }

    return false;
  };

  renderDescription = () => {
    const { selectedQuery } = this.props;
    if (selectedQuery) {
      return (
        <div className={`${baseClass}__description`}>
          <h2>Description</h2>
          <p>{selectedQuery.description || <>No description available.</>}</p>
        </div>
      );
    }

    return false;
  };

  render() {
    const { renderHeader, renderQuery, renderDescription } = this;
    const { onSelectQuery } = this.props;
    const { queryDropdownOptions } = this.state;

    return (
      <div className={baseClass}>
        {renderHeader()}
        <Dropdown
          options={queryDropdownOptions}
          onChange={onSelectQuery}
          placeholder={"Select query"}
        />
        {renderQuery()}
        {renderDescription()}
      </div>
    );
  }
}

export default SearchPackQuery;
