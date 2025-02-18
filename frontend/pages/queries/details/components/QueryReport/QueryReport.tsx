import React, { useState, useContext, useEffect, useCallback } from "react";

import { Row, Column } from "react-table";
import FileSaver from "file-saver";
import { QueryContext } from "context/query";

import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";
import { IQueryReport, IQueryReportResultRow } from "interfaces/query_report";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { generateResultsCountText } from "components/TableContainer/utilities/TableContainerUtils";
import TooltipWrapper from "components/TooltipWrapper";
import EmptyTable from "components/EmptyTable";

import generateReportColumnConfigsFromResults from "./QueryReportTableConfig";

interface IQueryReportProps {
  queryReport?: IQueryReport;
  isClipped?: boolean;
}

const baseClass = "query-report";
const CSV_TITLE = "Query";

const flattenResults = (results: IQueryReportResultRow[]) => {
  return results.map((result: IQueryReportResultRow) => {
    const hostInfoColumns = {
      host_display_name: result.host_name,
      last_fetched: result.last_fetched,
    };

    // hostInfoColumns displays the host metadata that is returned with every query
    // result.columns are the variable columns returned by the API that differ per query
    return { ...hostInfoColumns, ...result.columns };
  });
};

const QueryReport = ({
  queryReport,
  isClipped,
}: IQueryReportProps): JSX.Element => {
  const { lastEditedQueryName, lastEditedQueryBody } = useContext(QueryContext);

  const [filteredResults, setFilteredResults] = useState<Row[]>(
    flattenResults(queryReport?.results || [])
  );
  const [columnConfigs, setColumnConfigs] = useState<Column[]>([]);

  useEffect(() => {
    if (queryReport && queryReport.results && queryReport.results.length > 0) {
      const newColumnConfigs = generateReportColumnConfigsFromResults(
        flattenResults(queryReport.results)
      );

      // Update tableHeaders if new headers are found
      if (newColumnConfigs !== columnConfigs) {
        setColumnConfigs(newColumnConfigs);
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
        columnConfigs
      )
    );
  };

  const renderTableButtons = () => {
    return (
      <div className={`${baseClass}__results-cta`}>
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

  const renderResultsCount = useCallback(() => {
    const count = filteredResults.length;

    if (isClipped) {
      return (
        <>
          <TooltipWrapper
            tipContent={
              <>
                Fleet has retained a sample of early results for reference.
                Reporting is paused until existing data is deleted. <br />
                <br />
                You can reset this report by updating the query&apos;s SQL, or
                by temporarily enabling the <b>discard data</b> setting and
                disabling it again.
              </>
            }
          >
            {generateResultsCountText("results", count)}
          </TooltipWrapper>
        </>
      );
    }

    return <TableCount name="results" count={count} />;
  }, [filteredResults.length, isClipped]);

  const renderTable = () => {
    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          columnConfigs={columnConfigs}
          data={flattenResults(queryReport?.results || [])}
          // All empty states are handled in QueryDetailsPage.tsx and returned in lieu of QueryReport.tsx
          emptyComponent={() => {
            return (
              <EmptyTable
                className={baseClass}
                graphicName="empty-software"
                header="Nothing to report yet"
                info="This query has returned no data so far."
              />
            );
          }}
          defaultSortHeader="last_fetched"
          isLoading={false}
          isClientSidePagination
          isClientSideFilter
          isMultiColumnFilter
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle="results"
          customControl={() => renderTableButtons()}
          setExportRows={setFilteredResults}
          renderCount={renderResultsCount}
        />
      </div>
    );
  };

  return <div className={`${baseClass}__wrapper`}>{renderTable()}</div>;
};

export default QueryReport;
