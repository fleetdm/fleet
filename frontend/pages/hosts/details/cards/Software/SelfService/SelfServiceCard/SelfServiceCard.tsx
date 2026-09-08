import React, { useCallback, useEffect, useMemo } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { ISelfServiceCategory } from "interfaces/self_service_category";
import selfServiceCategoriesAPI, {
  ISelfServiceCategoriesResponse,
} from "services/entities/self_service_categories";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getPathWithQueryParams } from "utilities/url";

import Card from "components/Card";
import EmptyState from "components/EmptyState";
import Spinner from "components/Spinner";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import InstallAllInCategoryButton from "../components/InstallAllInCategoryButton";
import SelfServiceFilters from "../components/SelfServiceFilters";
import SelfServiceHeader from "../components/SelfServiceHeader";
import SelfServiceTable from "../components/SelfServiceTable";
import SelfServiceTiles from "../components/SelfServiceTiles";
import {
  countUninstalledForInstallAll,
  filterCategoriesWithSoftware,
  filterSoftwareByCustomCategory,
  filterSoftwareByQuery,
  hasInProgressInstallAllItems,
} from "../helpers";

const baseClass = "software-self-service";

export interface SelfServiceQueryParams {
  query: string;
  category_id?: number;
  order_key: string;
  order_direction: "asc" | "desc";
  page: number;
  per_page: number;
}

export interface ISelfServiceCardProps {
  contactUrl: string;
  deviceToken: string;
  queryParams: SelfServiceQueryParams;
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  selfServiceData?: IGetDeviceSoftwareResponse;
  tableConfig?: any;
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
  isEmpty: boolean;
  router: InjectedRouter;
  pathname: string;
  isMobileView?: boolean;
  onClickInstallAction: (
    softwareId: number,
    isScriptPackage?: boolean
  ) => Promise<boolean> | void;
  onInstallAllSuccess?: () => void;
}

const SelfServiceCard = ({
  contactUrl,
  deviceToken,
  queryParams,
  enhancedSoftware,
  selfServiceData,
  tableConfig,
  isLoading,
  isError,
  isFetching,
  isEmpty,
  router,
  pathname,
  isMobileView,
  onClickInstallAction,
  onInstallAllSuccess,
}: ISelfServiceCardProps) => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";

  // Device-token-scoped categories: the BE derives the fleet from the token
  // so the dropdown reflects this host's fleet, not the global fleet_id=0 set.
  // The queryKey's second element must match the queryFn arg to avoid
  // cross-device cache bleed.
  const { data: categoriesData, isSuccess: isCategoriesSuccess } = useQuery<
    ISelfServiceCategoriesResponse,
    Error,
    ISelfServiceCategory[]
  >(
    ["device_self_service_categories", deviceToken],
    () => selfServiceCategoriesAPI.getDeviceCategories(deviceToken),
    {
      select: (response) => response.self_service_categories,
      staleTime: 60_000,
    }
  );

  const categories = useMemo(() => categoriesData ?? [], [categoriesData]);

  // Hide categories with no software. enhancedSoftware is the host's full
  // self-service list (unpaginated), so everything downstream keys off this.
  const visibleCategories = useMemo(
    () => filterCategoriesWithSoftware(categories, enhancedSoftware),
    [categories, enhancedSoftware]
  );

  const softwareInSelectedCategory = useMemo(
    () =>
      filterSoftwareByCustomCategory(
        enhancedSoftware,
        visibleCategories,
        queryParams.category_id
      ),
    [enhancedSoftware, visibleCategories, queryParams.category_id]
  );

  // Trim the URL-supplied search once here so the desktop table filter, mobile
  // list, install-all count, and install-all POST all share identical
  // semantics. Without this, a deep-linked or trailing-space query like
  // `?query=%20fox%20` would leave react-table matching the raw value while
  // the helper/API used the trimmed one, contradicting the on-screen count.
  const normalizedQuery = queryParams.query?.trim() ?? "";

  // The install-all button count and target must match what's on screen. Layer
  // the search filter on top of the category filter so `uninstalledCount` and
  // the request sent to install_all both reflect the filtered subset.
  const softwareInSelectedCategoryMatchingQuery = useMemo(
    () => filterSoftwareByQuery(softwareInSelectedCategory, normalizedQuery),
    [softwareInSelectedCategory, normalizedQuery]
  );

  const uninstalledCount = useMemo(
    () =>
      countUninstalledForInstallAll(softwareInSelectedCategoryMatchingQuery),
    [softwareInSelectedCategoryMatchingQuery]
  );

  const hasInProgress = useMemo(
    () => hasInProgressInstallAllItems(softwareInSelectedCategoryMatchingQuery),
    [softwareInSelectedCategoryMatchingQuery]
  );

  const onClientSidePaginationChange = useCallback(
    (page: number) => {
      router.push(
        getPathWithQueryParams(pathname, {
          query: queryParams.query,
          category_id: queryParams.category_id,
          order_key: initialSortHeader,
          order_direction: initialSortDirection,
          page,
        })
      );
    },
    [
      pathname,
      queryParams.query,
      queryParams.category_id,
      initialSortDirection,
      initialSortHeader,
      router,
    ]
  );

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
        category_id: queryParams.category_id,
        order_key: initialSortHeader,
        order_direction: initialSortDirection,
        page: 0,
      })
    );
  };

  const onSortChange = ({ sortHeader, sortDirection }: ITableQueryData) => {
    router.push(
      getPathWithQueryParams(pathname, {
        ...queryParams,
        order_key: sortHeader,
        order_direction: sortDirection,
        query: queryParams.query !== undefined ? queryParams.query : undefined,
        category_id:
          queryParams.category_id !== undefined
            ? queryParams.category_id
            : undefined,
        page: 0,
      })
    );
  };

  const onCategoryChange = useCallback(
    (categoryId: number | undefined) => {
      router.push(
        getPathWithQueryParams(pathname, {
          category_id: categoryId,
          query: queryParams.query,
          order_key: initialSortHeader,
          order_direction: initialSortDirection,
          page: 0,
        })
      );
    },
    [
      pathname,
      queryParams.query,
      initialSortHeader,
      initialSortDirection,
      router,
    ]
  );

  // Recover from stale links: if the URL has a category_id that doesn't match
  // any visible category (admin deleted it, the list resolved empty, or the
  // category no longer has any self-service software), the trigger label would
  // fall through to "All" while filterSoftwareByCustomCategory returns [] —
  // contradicting what the label promises. Drop the param so the user lands
  // back on a real "All" view.
  useEffect(() => {
    // Wait for software too, else a valid category_id is cleared mid-load.
    if (
      !isCategoriesSuccess ||
      !selfServiceData ||
      queryParams.category_id === undefined
    )
      return;
    const idIsKnown = visibleCategories.some(
      (c) => c.id === queryParams.category_id
    );
    if (!idIsKnown) {
      onCategoryChange(undefined);
    }
  }, [
    isCategoriesSuccess,
    selfServiceData,
    visibleCategories,
    queryParams.category_id,
    onCategoryChange,
  ]);

  if (isLoading)
    return <Spinner {...(isMobileView && { variant: "mobile" })} />;
  if (isError)
    return (
      <EmptyState
        header="Error loading software"
        {...(isMobileView && { variant: "list" })}
      />
    );

  if ((isEmpty || !selfServiceData) && !isFetching) {
    return (
      <EmptyState
        header="No self-service software available yet"
        info="Your organization didn’t add any self-service software."
        {...(isMobileView && { variant: "list" })}
      />
    );
  }

  // Filter at this layer for both desktop and mobile. Two reasons: (1) the match
  // spans name, bundle_identifier, and custom display_name (the same columns the
  // backend MatchQuery searches), and TableContainer's built-in searchQueryColumn
  // is single-column, so we pre-filter here to widen it. (2) the empty state
  // stays in sync with the current search query. TableContainer's client-side
  // filter is debounced separately from the search field and briefly reported
  // the previous zero-result count when the URL query changed.
  const filteredSoftware = softwareInSelectedCategoryMatchingQuery;

  // The button is shown on desktop ONLY when a specific category is selected
  // (`category_id` is defined). On the unfiltered "All" view we suppress it so a
  // single click can't queue an install of the entire catalog — see #48485.
  // Visibility beyond this (count / in-progress / disabled state) is owned by
  // InstallAllInCategoryButton — see #47855 for the full rules.
  const installAllButton =
    !isMobileView && queryParams.category_id !== undefined ? (
      <InstallAllInCategoryButton
        uninstalledCount={uninstalledCount}
        hasInProgressInCategory={hasInProgress}
        deviceToken={deviceToken}
        categoryId={queryParams.category_id}
        query={normalizedQuery}
        onSuccess={() => onInstallAllSuccess?.()}
      />
    ) : null;

  if (isMobileView) {
    return (
      <>
        <SelfServiceHeader contactUrl={contactUrl} variant="mobile-header" />
        <div className={`${baseClass}__mobile-installers`}>
          <SelfServiceFilters
            query={queryParams.query}
            categoryId={queryParams.category_id}
            categories={visibleCategories}
            onSearchQueryChange={onSearchQueryChange}
            onCategoryChange={onCategoryChange}
          />
          <SelfServiceTiles
            contactUrl={contactUrl}
            enhancedSoftware={filteredSoftware}
            isFetching={isFetching}
            isEmptySearch={
              enhancedSoftware.length > 0 &&
              filteredSoftware.length === 0 &&
              !!queryParams.query
            }
            isEmptyCategory={
              enhancedSoftware.length > 0 &&
              filteredSoftware.length === 0 &&
              !queryParams.query &&
              queryParams.category_id !== undefined
            }
            onClickInstallAction={onClickInstallAction}
          />
        </div>
      </>
    );
  }
  return (
    <Card
      className={`${baseClass}__self-service-card`}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <SelfServiceHeader contactUrl={contactUrl} />
      <div className={`${baseClass}__content`}>
        <SelfServiceFilters
          query={queryParams.query}
          categoryId={queryParams.category_id}
          categories={visibleCategories}
          onSearchQueryChange={onSearchQueryChange}
          onCategoryChange={onCategoryChange}
          installAllSlot={installAllButton}
        />
        <SelfServiceTable
          baseClass={baseClass}
          contactUrl={contactUrl}
          queryParams={{ ...queryParams, query: normalizedQuery }}
          enhancedSoftware={filteredSoftware}
          selfServiceData={selfServiceData}
          tableConfig={tableConfig}
          isFetching={isFetching}
          onSortChange={onSortChange}
          onClientSidePaginationChange={onClientSidePaginationChange}
        />
      </div>
    </Card>
  );
};

export default SelfServiceCard;
