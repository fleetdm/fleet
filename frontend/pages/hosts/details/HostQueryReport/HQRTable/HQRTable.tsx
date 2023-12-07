import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon";
import TableContainer from "components/TableContainer";
import React, { useCallback, useState } from "react";
import { Row } from "react-table";
import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";
import FileSaver from "file-saver";
import generateColumnConfigs from "./HQRTableConfig";

const baseClass = "hqr-table";

interface IHQRTable {
  queryName?: string;
  hostName?: string;
  rows: Record<string, any>[];
  reportClipped?: boolean;
  lastFetched?: string | null; // timestamp
  onShowQuery: () => void;
  isLoading: boolean;
}

const DEFAULT_CSV_TITLE = "Host-Specific Query Report";

const HQRTable = ({
  queryName,
  hostName,
  rows,
  reportClipped,
  lastFetched,
  onShowQuery,
  isLoading,
}: IHQRTable) => {
  const [filteredResults, setFilteredResults] = useState<Row[]>([]);

  const columnConfigs = generateColumnConfigs(rows);

  const renderTableButtons = useCallback(() => {
    const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
      evt.preventDefault();
      FileSaver.saveAs(
        generateCSVQueryResults(
          filteredResults,
          generateCSVFilename(
            queryName && hostName
              ? `'${queryName}' query report results for host '${hostName}'`
              : DEFAULT_CSV_TITLE
          ),
          columnConfigs
        )
      );
    };
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQuery}
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
  }, [onShowQuery, filteredResults, queryName, hostName, columnConfigs]);

  const emptyComponent = () => {
    // if query hasn't run on this host
    // rows.length === 0

    if (!lastFetched) {
      // collecting results
      return (
        <EmptyTable
          className={`${baseClass}__collecting-results`}
          graphicName="collecting-results"
          header="Collecting results..."
          info={`Fleet is collecting query results from ${hostName}. Check back later.`}
        />
      );
    }
    if (reportClipped) {
      return (
        <EmptyTable
          className={`${baseClass}__report-clipped`}
          graphicName="empty-software"
          header="Report clipped"
          info="This query has paused reporting in Fleet, and no results were saved for this host."
        />
      );
    }
    return (
      // nothing to report
      <EmptyTable
        className={`${baseClass}__nothing-to-report`}
        graphicName="empty-software"
        header="Nothing to report"
        info={`This query has run on ${hostName}, but returned no data for this host.`}
      />
    );
  };

  const renderCount = () => {
    const count = filteredResults.length;
    return (
      <div className={`${baseClass}__count `}>
        <span>{`${count} result${count === 1 ? "" : "s"}`}</span>
      </div>
    );
  };

  return (
    <TableContainer
      isClientSidePagination
      isClientSideFilter
      isMultiColumnFilter
      showMarkAllPages={false}
      isAllPagesSelected={false}
      resultsTitle="results"
      customControl={renderTableButtons}
      setExportRows={setFilteredResults}
      {...{ columnConfigs, data: rows, emptyComponent, isLoading, renderCount }}
    />
  );
};

export default HQRTable;
