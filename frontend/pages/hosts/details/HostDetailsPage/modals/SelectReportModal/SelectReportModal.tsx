import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { filter, includes } from "lodash";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import permissions from "utilities/permissions";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";

import queryAPI from "services/entities/queries";

// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";

import {
  IListQueriesResponse,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { API_ALL_TEAMS_ID } from "interfaces/team";
import { DEFAULT_TARGETS_BY_TYPE } from "interfaces/target";
import { getPathWithQueryParams } from "utilities/url";

export interface ISelectReportModalProps {
  onCancel: () => void;
  isOnlyObserver?: boolean;
  hostId: number;
  hostTeamId: number | null;
  router: InjectedRouter; // v3
  currentTeamId: number | undefined;
}

const baseClass = "select-report-modal";

const SelectReportModal = ({
  onCancel,
  isOnlyObserver,
  hostId,
  hostTeamId,
  router,
  currentTeamId,
}: ISelectReportModalProps): JSX.Element => {
  const { setSelectedQueryTargetsByType } = useContext(QueryContext);

  const { data: reports, error: reportsErr } = useQuery<
    IListQueriesResponse,
    Error,
    ISchedulableQuery[],
    IQueryKeyQueriesLoadAll[]
  >(
    [
      {
        scope: "queries",
        teamId: hostTeamId || API_ALL_TEAMS_ID,
        mergeInherited: hostTeamId !== API_ALL_TEAMS_ID,
      },
    ],
    ({ queryKey }) => queryAPI.loadAll(queryKey[0]),
    {
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IListQueriesResponse) => data.queries,
    }
  );

  const onRunCustomReport = () => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      getPathWithQueryParams(PATHS.NEW_REPORT, {
        host_id: hostId,
        fleet_id: currentTeamId,
      })
    );
  };

  const onRunSavedReport = (selectedReport: ISchedulableQuery) => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      getPathWithQueryParams(PATHS.EDIT_REPORT(selectedReport.id), {
        host_id: hostId,
        fleet_id: currentTeamId,
      })
    );
  };

  let reportsAvailableToRun = reports;

  const { currentUser, isObserverPlus } = useContext(AppContext);

  /*  Context team id might be different that host's team id
  Observer plus must be checked against host's team id  */
  const isHostsTeamObserverPlus = currentUser
    ? permissions.isObserverPlus(currentUser, hostTeamId)
    : false;

  const canCreateReport =
    !isOnlyObserver || isObserverPlus || isHostsTeamObserverPlus;

  const [reportsFilter, setReportsFilter] = useState("");

  if (isOnlyObserver && !isObserverPlus && !isHostsTeamObserverPlus) {
    reportsAvailableToRun =
      reports?.filter((report) => report.observer_can_run === true) || [];
  }

  const getReports = () => {
    if (!reportsFilter) {
      return reportsAvailableToRun;
    }

    const lowerReportFilter = reportsFilter.toLowerCase();

    return filter(reportsAvailableToRun, (report) => {
      if (!report.name) {
        return false;
      }

      const lowerReportName = report.name.toLowerCase();

      return includes(lowerReportName, lowerReportFilter);
    });
  };

  const onFilterReports = useCallback(
    (filterString: string): void => {
      setReportsFilter(filterString);
    },
    [setReportsFilter]
  );

  const reportsFiltered = getReports();

  const reportsCount = reportsFiltered?.length || 0;

  const renderDescription = (): JSX.Element => {
    return (
      <div className={`${baseClass}__description`}>
        Choose a report to run on this host
        {canCreateReport && (
          <>
            {" "}
            or{" "}
            <Button variant="link" onClick={onRunCustomReport}>
              create a report
            </Button>
          </>
        )}
        .
      </div>
    );
  };

  const renderReports = (): JSX.Element => {
    if (reportsErr) {
      return <DataError />;
    }

    if (!reportsFilter && reportsCount === 0) {
      return (
        <EmptyState
          variant="list"
          header="No saved reports"
          info={
            canCreateReport ? (
              <>
                <Button variant="link" onClick={onRunCustomReport}>
                  Create a report
                </Button>{" "}
                to run.
              </>
            ) : (
              "No reports are available to run."
            )
          }
        />
      );
    }

    if (reportsCount > 0) {
      const reportList =
        reportsFiltered?.map((report) => {
          return (
            <Button
              key={report.id}
              variant="unstyled-modal-query"
              className={`${baseClass}__modal-query-button`}
              onClick={() => onRunSavedReport(report)}
            >
              <>
                <span className="info__header">{report.name}</span>
                {report.description && (
                  <span className="info__data">{report.description}</span>
                )}
              </>
            </Button>
          );
        }) || [];

      return (
        <>
          <InputFieldWithIcon
            name="report-filter"
            onChange={onFilterReports}
            placeholder="Filter reports"
            value={reportsFilter}
            autofocus
            iconSvg="search"
          />
          <div className={`${baseClass}__report-selection`}>{reportList}</div>
        </>
      );
    }

    if (reportsFilter && reportsCount === 0) {
      return (
        <>
          <div className={`${baseClass}__filter-queries`}>
            <InputFieldWithIcon
              name="report-filter"
              onChange={onFilterReports}
              placeholder="Filter reports"
              value={reportsFilter}
              autofocus
              iconSvg="search"
            />
          </div>
          <EmptyState
            variant="list"
            header="No reports match the current search criteria"
            info="Expecting to see reports? Try again in a few seconds as the system catches up."
          />
        </>
      );
    }
    return <></>;
  };

  return (
    <Modal
      title="Select a report"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="large"
    >
      {renderDescription()}
      {renderReports()}
      <div className="modal-cta-wrap">
        <Button onClick={onCancel} variant="inverse">
          Close
        </Button>
      </div>
    </Modal>
  );
};

export default SelectReportModal;
