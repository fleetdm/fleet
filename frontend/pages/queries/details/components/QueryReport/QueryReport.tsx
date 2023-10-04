import React, { useState, useContext, useEffect } from "react";
// import { Row, Column } from "react-table";

// import classnames from "classnames";
import FileSaver from "file-saver";
import { QueryContext } from "context/query";

import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";
import { IQueryReport } from "interfaces/query_report";
import { humanLastSeen } from "utilities/helpers";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TableContainer from "components/TableContainer";
import ShowQueryModal from "components/modals/ShowQueryModal";

import generateResultsTableHeaders from "./QueryReportTableConfig";

interface IQueryReportProps {
  queryReport?: IQueryReport;
}

const baseClass = "query-report";
const CSV_TITLE = "Query";

/*
I want to return 

[
  {
    host_name: "foo",
    last_fetched: "2021-01-19T17:08:31Z",
    model: "USB 2.0 Hub",
    vendor: "VIA Labs, Inc.",
  },
  {
    host_name: "foo",
    last_fetched: "2021-01-19T17:08:31Z",
    model: "USB Keyboard",
    vendor: "VIA Labs, Inc.",
  },
]
*/
const uiResults = (results: any) => {
  return results.map((result: any) => {
    const obj = {
      host_id: result.host_id,
      host_name: result.host_name,
      last_fetched: humanLastSeen(result.last_fetched),
    };
    // Object.keys(result.columns).forEach(key => {
    //   obj[key] = result.columns[key];
    // })
    // Object.assign(obj, result.columns);
    // console.log("obj", obj);
    // console.log("Object.keys(obj)", Object.keys(obj));
    // return Object.values(obj);
    console.log("obj", obj);
    return obj;
  });
};

const QueryReport = ({ queryReport }: IQueryReportProps): JSX.Element => {
  const { lastEditedQueryName, lastEditedQueryBody } = useContext(QueryContext);

  const [showQueryModal, setShowQueryModal] = useState(false);
  const [filteredResults, setFilteredResults] = useState<any>(
    uiResults(queryReport?.results)
  );
  const [tableHeaders, setTableHeaders] = useState<any>([]);
  const [queryResultsForTableRender, setQueryResultsForTableRender] = useState(
    queryReport?.results
  );

  useEffect(() => {
    if (queryReport && queryReport.results && queryReport.results.length > 0) {
      const generatedTableHeaders = generateResultsTableHeaders(
        queryReport.results
      );
      // Update tableHeaders if new headers are found
      if (generatedTableHeaders !== tableHeaders) {
        setTableHeaders(generatedTableHeaders);
      }
    }
  }, [queryReport]); // Cannot use tableHeaders as it will cause infinite loop with setTableHeaders

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    FileSaver.saveAs(
      generateCSVQueryResults(
        filteredResults,
        generateCSVFilename(`${lastEditedQueryName || CSV_TITLE} - Results`),
        tableHeaders
      )
    );
  };

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const renderNoResults = () => {
    return <p className="no-results-message">TODO</p>;
  };

  const renderTableButtons = () => {
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="text-icon"
        >
          <>
            Show query <Icon name="eye" />
          </>
        </Button>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={onExportQueryResults}
          variant="text-icon"
        >
          <>
            Export results
            <Icon name="download" color="core-fleet-blue" />
          </>
        </Button>
      </div>
    );
  };

  console.log("tableHeaders", tableHeaders);
  console.log("filteredResults", filteredResults);
  const renderTable = () => {
    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          columns={tableHeaders}
          data={filteredResults}
          emptyComponent={renderNoResults}
          isLoading={false}
          isClientSidePagination
          isClientSideFilter
          isMultiColumnFilter
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle="results"
          customControl={() => renderTableButtons()}
          // setExportRows={setFilteredResults}
        />
      </div>
    );
  };

  return (
    <div className={`${baseClass}__wrapper`}>
      {renderTable()}
      {showQueryModal && (
        <ShowQueryModal
          query={lastEditedQueryBody}
          onCancel={onShowQueryModal}
        />
      )}
    </div>
  );
};

export default QueryReport;
