import React, { useCallback, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { Row } from "react-table";
import { AxiosError } from "axios";
import classnames from "classnames";

import { notify } from "components/ToastNotification";
import {
  IPolicy,
  IPolicyAutomationActivity,
  OtherAutomationType,
} from "interfaces/policy";
import policiesAPI, {
  IGetPolicyAutomationActivitiesParams,
  IPolicyAutomationActivitiesResponse,
  PolicyAutomationActivitiesOrderKey,
} from "services/entities/policies";
import { OrderDirection } from "services/entities/common";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { pluralize } from "utilities/strings/stringUtils";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyState from "components/EmptyState";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import generateColumnConfigs from "./PolicyAutomationsActivitiesTableConfig";
import PolicyAutomationActivityDetailsModal from "../PolicyAutomationActivityDetailsModal";
import PolicyResetModal from "../PolicyResetModal";

const baseClass = "policy-automations-activities-table";

const DEFAULT_PAGE_SIZE = 50;
const DEFAULT_SORT_HEADER: PolicyAutomationActivitiesOrderKey = "created_at";
const DEFAULT_SORT_DIRECTION: OrderDirection = "desc";

type StatusFilter = NonNullable<IGetPolicyAutomationActivitiesParams["status"]>;

const STATUS_FILTER_OPTIONS: CustomOptionType[] = [
  { label: "All", value: "" },
  { label: "Successful", value: "success" },
  { label: "Failed", value: "error" },
];

interface IPolicyAutomationsActivitiesTableProps {
  policy: IPolicy;
  currentAutomatedPolicies: number[];
  otherAutomationType?: OtherAutomationType;
  canResetPolicy: boolean;
}

const PolicyAutomationsActivitiesTable = ({
  policy,
  currentAutomatedPolicies,
  otherAutomationType,
  canResetPolicy,
}: IPolicyAutomationsActivitiesTableProps): JSX.Element => {
  const { id: policyId } = policy;
  const queryClient = useQueryClient();

  const [page, setPage] = useState(0);
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [
    sortHeader,
    setSortHeader,
  ] = useState<PolicyAutomationActivitiesOrderKey>(DEFAULT_SORT_HEADER);
  const [sortDirection, setSortDirection] = useState<OrderDirection>(
    DEFAULT_SORT_DIRECTION
  );

  const [
    selectedActivity,
    setSelectedActivity,
  ] = useState<IPolicyAutomationActivity | null>(null);
  const [showResetModal, setShowResetModal] = useState(false);
  const [resetHostDisplayName, setResetHostDisplayName] = useState<
    string | undefined
  >(undefined);

  const { data, isLoading, isError } = useQuery<
    IPolicyAutomationActivitiesResponse,
    AxiosError
  >(
    [
      "policyAutomationActivities",
      policyId,
      page,
      DEFAULT_PAGE_SIZE,
      sortHeader,
      sortDirection,
      searchQuery,
      statusFilter,
    ],
    () =>
      policiesAPI.getAutomationActivities({
        policyId,
        page,
        perPage: DEFAULT_PAGE_SIZE,
        orderKey: sortHeader,
        orderDirection: sortDirection,
        query: searchQuery,
        status: statusFilter,
      }),
    { ...DEFAULT_USE_QUERY_OPTIONS, keepPreviousData: true }
  );

  const { mutateAsync: resetPolicy, isLoading: isResetting } = useMutation(
    () => policiesAPI.reset(policyId),
    {
      onSuccess: () => {
        notify.success("Policy reset successfully.");
        queryClient.invalidateQueries(["policyAutomationActivities", policyId]);
        queryClient.invalidateQueries(["policy", policyId]);
        setShowResetModal(false);
      },
      onError: (error) => {
        notify.error("Couldn't reset policy. Please try again.", {
          response: error,
        });
      },
    }
  );

  // Search lives in our own header (see below), so we only take sort/page from
  // the table here.
  const onQueryChange = useCallback((newTableQuery: ITableQueryData) => {
    const {
      pageIndex: newPage,
      sortHeader: newSortHeader,
      sortDirection: newSortDirection,
    } = newTableQuery;

    setSortHeader(newSortHeader as PolicyAutomationActivitiesOrderKey);
    setSortDirection(newSortDirection as OrderDirection);
    setPage(newPage);
  }, []);

  const onSearchChange = useCallback((value: string) => {
    setSearchQuery(value);
    setPage(0);
  }, []);

  const onStatusFilterChange = useCallback(
    (option: CustomOptionType | null) => {
      setStatusFilter((option?.value ?? "") as StatusFilter);
      setPage(0);
    },
    []
  );

  const onClickResetPolicy = useCallback(() => {
    setResetHostDisplayName(undefined);
    setShowResetModal(true);
  }, []);

  // The reset is always policy-wide; the host name is only used to make the
  // confirmation copy concrete when the reset is opened from a specific run.
  const onResetFromActivity = useCallback(() => {
    setResetHostDisplayName(selectedActivity?.host_display_name);
    setSelectedActivity(null);
    setShowResetModal(true);
  }, [selectedActivity]);

  const isFiltered = searchQuery !== "" || statusFilter !== "";

  const renderEmptyState = useCallback(() => {
    if (isFiltered) {
      return (
        <EmptyState
          header="No automation runs match your filters"
          info="Try changing your search or status filter."
        />
      );
    }
    return (
      <EmptyState
        header="No automation runs"
        info="When this policy's automations run, their results will appear here."
      />
    );
  }, [isFiltered]);

  const columnConfigs = useMemo(
    () => generateColumnConfigs(baseClass, setSelectedActivity),
    []
  );

  const count = data?.count ?? 0;
  // Hide the count and filter/search controls in the unfiltered empty state, so
  // a policy that has never run an automation shows just the reset action.
  const showControls = count > 0 || isFiltered;

  if (isError) {
    return <DataError description="Could not load automation runs." />;
  }

  return (
    <div className={baseClass}>
      <div
        className={classnames(`${baseClass}__header`, {
          [`${baseClass}__header--inline`]: !showControls,
        })}
      >
        <h2 className={`${baseClass}__title`}>Automation runs</h2>
        <div className={`${baseClass}__controls-row`}>
          {showControls && (
            <span className={`${baseClass}__count`}>
              {count} {pluralize(count, "run")}
            </span>
          )}
          <div className={`${baseClass}__controls`}>
            {canResetPolicy && (
              <Button variant="inverse" onClick={onClickResetPolicy}>
                Reset policy
                <Icon name="refresh" />
              </Button>
            )}
            {showControls && (
              <>
                <DropdownWrapper
                  name="automation-status-filter"
                  className={`${baseClass}__status-filter`}
                  options={STATUS_FILTER_OPTIONS}
                  value={statusFilter}
                  onChange={onStatusFilterChange}
                  variant="table-filter"
                  isSearchable={false}
                />
                <div className={`${baseClass}__search`}>
                  <SearchField
                    placeholder="Search hosts"
                    defaultValue={searchQuery}
                    onChange={onSearchChange}
                  />
                </div>
              </>
            )}
          </div>
        </div>
      </div>
      <TableContainer
        columnConfigs={columnConfigs}
        data={data?.activities ?? []}
        isLoading={isLoading}
        manualSortBy
        pageIndex={page}
        pageSize={DEFAULT_PAGE_SIZE}
        disableNextPage={!data?.meta.has_next_results}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        searchable={false}
        onQueryChange={onQueryChange}
        emptyComponent={renderEmptyState}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableMultiRowSelect
        onClickRow={(row: Row<IPolicyAutomationActivity>) =>
          setSelectedActivity(row.original)
        }
      />
      {selectedActivity && (
        <PolicyAutomationActivityDetailsModal
          activity={selectedActivity}
          onCancel={() => setSelectedActivity(null)}
          onResetPolicy={canResetPolicy ? onResetFromActivity : undefined}
        />
      )}
      {showResetModal && (
        <PolicyResetModal
          policy={policy}
          hostDisplayName={resetHostDisplayName}
          currentAutomatedPolicies={currentAutomatedPolicies}
          otherAutomationType={otherAutomationType}
          isResetting={isResetting}
          onSubmit={resetPolicy}
          onCancel={() => setShowResetModal(false)}
        />
      )}
    </div>
  );
};

export default PolicyAutomationsActivitiesTable;
