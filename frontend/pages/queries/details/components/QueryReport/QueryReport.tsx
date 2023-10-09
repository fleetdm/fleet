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

const tableResults = (results: any) => {
  return results.map((result: any) => {
    const hostInfo = {
      host_display_name: result.host_name,
      last_fetched: humanLastSeen(result.last_fetched),
    };

    const tableData = { ...hostInfo, ...result.columns };
    return tableData;
  });
};

const QueryReport = ({ queryReport }: IQueryReportProps): JSX.Element => {
  const { lastEditedQueryName, lastEditedQueryBody } = useContext(QueryContext);

  const [showQueryModal, setShowQueryModal] = useState(false);
  const [filteredResults, setFilteredResults] = useState<any>(
    tableResults(queryReport?.results)
  );
  const [tableHeaders, setTableHeaders] = useState<any>([]);

  useEffect(() => {
    if (queryReport && queryReport.results && queryReport.results.length > 0) {
      const generatedTableHeaders = generateResultsTableHeaders(
        tableResults(queryReport.results)
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
        generateCSVFilename(
          `${lastEditedQueryName || CSV_TITLE} - Query Report`
        ),
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

  const renderTable = () => {
    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          columns={tableHeaders}
          data={tableResults(queryReport?.results) || []}
          emptyComponent={renderNoResults}
          isLoading={false}
          isClientSidePagination
          isClientSideFilter
          isMultiColumnFilter
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle="results"
          customControl={() => renderTableButtons()}
          setExportRows={setFilteredResults}
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
