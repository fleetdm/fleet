import React, { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import { useDebouncedCallback } from "use-debounce";
import { isEmpty } from "lodash";

import { IHost } from "interfaces/host";
import { ILabelSummary } from "interfaces/label";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";
import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";
import labelsAPI from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import SearchField from "components/forms/fields/SearchField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

import {
  ANY_SEVERITY_VALUE,
  ISeverityFilterValue,
  severityFilters,
  SeverityValue,
} from "components/SeverityFilter";

import SoftwareFilters from "./SoftwareFilters";
import {
  isEpssActive,
  ISoftwareFilterErrors,
  NO_CATEGORIES_MSG,
  SoftwareFilterField,
  validateSoftwareFilters,
} from "./SoftwareFilters/helpers";

const baseClass = "chart-filter-modal";

export type ChartFilterTab = "hosts" | "software";

const HOSTS_TAB_INDEX = 0;
const SOFTWARE_TAB_INDEX = 1;

// Exported for testing.
export const PLATFORM_OPTIONS = [
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
  { label: "ChromeOS", value: "chrome" },
  { label: "iOS", value: "ios" },
  { label: "iPadOS", value: "ipados" },
  { label: "Android", value: "android" },
];

type HostFilterMode = "none" | "include" | "exclude";

export interface IChartFilterState {
  labelIDs: number[];
  platforms: string[];
  hostFilterMode: HostFilterMode;
  selectedHosts: IHost[];
  // Software (cve) filters. softwareFilters holds the checked category
  // values (defaults to all). epssMin/epssMax are raw 0–100 % strings ("" =
  // unset); the card converts them to the 0.0–1.0 API value.
  softwareFilters: string[];
  knownExploit: boolean;
  epssMin: string;
  epssMax: string;
  // Severity (CVSS). `severity` is the SeverityFilter option value; cvssMin /
  // cvssMax are the raw 0–10 score strings behind it, and they always mirror
  // the selection — a preset carries its own bounds, Any and Custom carry none.
  severity: SeverityValue;
  cvssMin: string;
  cvssMax: string;
  excludeCVEs: string[];
}

interface IChartFilterModalProps {
  filters: IChartFilterState;
  currentTeamId?: number;
  // metric drives whether the Software tab is shown (cve only).
  metric: string;
  initialTab?: ChartFilterTab;
  initialShowAdvanced?: boolean;
  onApply: (filters: IChartFilterState) => void;
  onCancel: () => void;
}

const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 300;

const ChartFilterModal = ({
  filters,
  currentTeamId,
  metric,
  initialTab = "hosts",
  initialShowAdvanced = false,
  onApply,
  onCancel,
}: IChartFilterModalProps): JSX.Element => {
  const isCVE = metric === "cve";
  const [activeTab, setActiveTab] = useState(
    initialTab === "software" ? SOFTWARE_TAB_INDEX : HOSTS_TAB_INDEX
  );

  // Software (cve) filter state.
  const [softwareFilters, setSoftwareFilters] = useState<string[]>(
    filters.softwareFilters
  );
  const [knownExploit, setKnownExploit] = useState<boolean>(
    filters.knownExploit
  );
  const [epssMin, setEpssMin] = useState<string>(filters.epssMin);
  const [epssMax, setEpssMax] = useState<string>(filters.epssMax);
  const [severityFilter, setSeverityFilter] = useState<ISeverityFilterValue>({
    severity: filters.severity,
    minScore: filters.cvssMin,
    maxScore: filters.cvssMax,
  });
  const [excludeCVEs, setExcludeCVEs] = useState<string[]>(filters.excludeCVEs);

  // Only the errors currently being shown — never everything that is invalid.
  // See frontend/docs/patterns.md#data-validation.
  const [softwareErrors, setSoftwareErrors] = useState<ISoftwareFilterErrors>(
    {}
  );
  // A field is dirty once the user has edited it. Seeded values that happen to
  // be invalid must not error on a mere tab through. A ref because it gates
  // handlers only.
  const dirtyFields = useRef(new Set<SoftwareFilterField>());

  const [selectedLabelIDs, setSelectedLabelIDs] = useState<number[]>(
    filters.labelIDs
  );
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(
    filters.platforms
  );
  // Host filter mode is either "include" or "exclude", used when selecting
  // individual hosts to filter on.
  const [hostFilterMode, setHostFilterMode] = useState<HostFilterMode>(
    filters.hostFilterMode === "none" ? "exclude" : filters.hostFilterMode
  );
  // Individual hosts selected for filtering.
  const [selectedHosts, setSelectedHosts] = useState<IHost[]>(
    filters.selectedHosts
  );
  const [searchInput, setSearchInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [pageCount, setPageCount] = useState(1);
  const [searchFieldKey, setSearchFieldKey] = useState(0);

  const listRef = useRef<HTMLDivElement>(null);
  const selectedHostIds = new Set(selectedHosts.map((h) => h.id));

  const debouncedSetSearchQuery = useDebouncedCallback((value: string) => {
    setSearchQuery(value);
    setPageCount(1);
    if (listRef.current) {
      listRef.current.scrollTop = 0;
    }
  }, SEARCH_DEBOUNCE_MS);

  // Flush pending debounced call on unmount so it doesn't fire after teardown.
  useEffect(() => {
    return () => debouncedSetSearchQuery.cancel();
  }, [debouncedSetSearchQuery]);

  // Fetch hosts with pagination — load all pages up to pageCount.
  // Note that we use infinite scrolling in the UI, rather than
  // traditional pagination controls, so we keep previously loaded
  // pages in the cache and just increase the page count as the user scrolls.
  const {
    data: hostsData,
    isLoading: isLoadingHosts,
    error: hostsError,
  } = useQuery<ILoadHostsResponse, Error>(
    ["chartFilterHosts", currentTeamId, searchQuery, pageCount],
    () =>
      hostsAPI.loadHosts({
        page: 0,
        perPage: pageCount * PAGE_SIZE,
        teamId: currentTeamId,
        globalFilter: searchQuery || undefined,
        sortBy: [{ key: "display_name", direction: "asc" }],
      }),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  const hosts = hostsData?.hosts ?? [];
  const hasMore = hosts.length === pageCount * PAGE_SIZE;

  // This implements "infinite" scrolling by increasing the page count when the user scrolls
  // near the bottom of the list.
  const handleScroll = useCallback(() => {
    const el = listRef.current;
    if (!el || !hasMore || isLoadingHosts) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 40) {
      setPageCount((prev) => prev + 1);
    }
  }, [hasMore, isLoadingHosts]);

  const handleSearchChange = useCallback(
    (value: string) => {
      setSearchInput(value);
      debouncedSetSearchQuery(value);
    },
    [debouncedSetSearchQuery]
  );

  const { data: labels } = useQuery<ILabelSummary[]>(
    ["labelsSummary", currentTeamId],
    () => labelsAPI.summary(currentTeamId ?? null).then((res) => res.labels),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 60000,
    }
  );

  const labelOptions = (labels || [])
    .filter((l) => l.label_type !== "builtin")
    .map((l) => ({
      label: l.name,
      value: l.id,
    }));

  const softwareFormData = {
    categories: softwareFilters,
    epssMin,
    epssMax,
    minScore: severityFilter.minScore,
    maxScore: severityFilter.maxScore,
  };

  const markDirty = (field: SoftwareFilterField) => {
    dirtyFields.current.add(field);
  };

  // Blur validates that one field and nothing else.
  const onFieldBlur = (field: SoftwareFilterField) => {
    if (!dirtyFields.current.has(field)) {
      return;
    }
    const { [field]: fieldError } = validateSoftwareFilters(softwareFormData);
    setSoftwareErrors((prev) => ({ ...prev, [field]: fieldError }));
  };

  // Focus clears immediately so the label returns while the user edits.
  const onFieldFocus = (field: SoftwareFilterField) => {
    setSoftwareErrors((prev) =>
      prev[field] ? { ...prev, [field]: undefined } : prev
    );
  };

  const onChangeEpssMin = (value: string) => {
    markDirty("epssMin");
    setEpssMin(value);
  };

  const onChangeEpssMax = (value: string) => {
    markDirty("epssMax");
    setEpssMax(value);
  };

  const onChangeSeverityFilter = (next: ISeverityFilterValue) => {
    if (next.severity === severityFilter.severity) {
      // Same option means the user typed in a score input.
      if (next.minScore !== severityFilter.minScore) markDirty("cvssMin");
      if (next.maxScore !== severityFilter.maxScore) markDirty("cvssMax");
    } else {
      // Switching options rewrites both bounds, so any error on them no longer
      // describes anything.
      setSoftwareErrors((prev) => ({
        ...prev,
        cvssMin: undefined,
        cvssMax: undefined,
      }));
    }
    setSeverityFilter(next);
  };

  const onChangeCategories = (next: string[]) => {
    markDirty("categories");
    // A selection control has no blur to validate on, and going from one
    // category to none is unambiguous, so this one resolves on change.
    setSoftwareErrors((prev) => ({
      ...prev,
      categories: next.length === 0 ? NO_CATEGORIES_MSG : undefined,
    }));
    setSoftwareFilters(next);
  };

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Enter submits a native form from any field in it, and applying part-way
    // through typing a search would be a surprise. Clicking Apply moves focus to
    // the button, so a still-focused text input means Enter came from a
    // SearchField — ignore it. The number inputs keep Enter-to-apply.
    const focused = document.activeElement as HTMLInputElement | null;
    if (focused?.tagName === "INPUT" && focused.type === "text") {
      return;
    }
    // Submit is a checkpoint: validate everything regardless of dirty state,
    // show all the errors at once, and return without applying.
    if (isCVE) {
      const errors = validateSoftwareFilters(softwareFormData);
      if (Object.keys(errors).length > 0) {
        setSoftwareErrors(errors);
        // The errors are all on the Software tab, so show it.
        setActiveTab(SOFTWARE_TAB_INDEX);
        return;
      }
    }

    onApply({
      labelIDs: selectedLabelIDs,
      platforms: selectedPlatforms,
      hostFilterMode,
      selectedHosts,
      softwareFilters,
      knownExploit,
      epssMin,
      epssMax,
      severity: severityFilter.severity,
      cvssMin: severityFilter.minScore,
      cvssMax: severityFilter.maxScore,
      excludeCVEs,
    });
  };

  const handleClear = () => {
    setSelectedLabelIDs([]);
    setSelectedPlatforms([]);
    setHostFilterMode("none");
    setSelectedHosts([]);
    setSearchInput("");
    setSearchQuery("");
    setPageCount(1);
    setSearchFieldKey((k) => k + 1);
    debouncedSetSearchQuery.cancel();
    // Reset software filters to their defaults (all categories selected).
    setSoftwareFilters([...ALL_CVE_SOFTWARE_CATEGORY_VALUES]);
    setKnownExploit(false);
    setEpssMin("");
    setEpssMax("");
    // Clear all resets severity to Any, not to the Critical seed — otherwise
    // the filters would still be active and the Clear all button would never
    // disappear after being clicked.
    setSeverityFilter({
      severity: ANY_SEVERITY_VALUE,
      minScore: "",
      maxScore: "",
    });
    setExcludeCVEs([]);
    setSoftwareErrors({});
    dirtyFields.current.clear();
  };

  const handleTabChange = (index: number) => {
    const mode = index === 0 ? "exclude" : "include";
    setHostFilterMode(mode);
  };

  const toggleHost = (host: IHost) => {
    if (selectedHostIds.has(host.id)) {
      setSelectedHosts((prev) => prev.filter((h) => h.id !== host.id));
    } else {
      setSelectedHosts((prev) => [...prev, host]);
    }
  };

  const removeHost = (hostId: number) => {
    setSelectedHosts((prev) => prev.filter((h) => h.id !== hostId));
  };

  const softwareFiltersActive =
    isCVE &&
    (softwareFilters.length !== ALL_CVE_SOFTWARE_CATEGORY_VALUES.length ||
      knownExploit ||
      isEpssActive(epssMin, epssMax) ||
      !isEmpty(severityFilters(severityFilter)) ||
      excludeCVEs.length > 0);

  const hasFilters =
    selectedLabelIDs.length > 0 ||
    selectedPlatforms.length > 0 ||
    selectedHosts.length > 0 ||
    softwareFiltersActive;

  // Inner host include/exclude tab.
  const tabIndex = hostFilterMode === "include" ? 1 : 0;

  const renderHostSearch = () => (
    <div className={`${baseClass}__host-search`}>
      <SearchField
        key={searchFieldKey}
        placeholder="Search name, hostname, or serial number"
        defaultValue={searchInput}
        onChange={handleSearchChange}
      />
      {selectedHosts.length > 0 && (
        <div className={`${baseClass}__pills`}>
          {selectedHosts.map((host) => (
            <button
              key={host.id}
              type="button"
              className={`${baseClass}__pill`}
              onClick={() => removeHost(host.id)}
            >
              {host.display_name}
              <Icon name="close" />
            </button>
          ))}
        </div>
      )}
      <div
        className={`${baseClass}__results-list`}
        ref={listRef}
        onScroll={handleScroll}
      >
        {hosts.map((host) => (
          <div key={host.id} className={`${baseClass}__results-row`}>
            <Checkbox
              name={`host-${host.id}`}
              value={selectedHostIds.has(host.id)}
              onChange={() => toggleHost(host)}
            >
              {host.display_name}
            </Checkbox>
          </div>
        ))}
        {hostsError && (
          <div className={`${baseClass}__results-status`} role="alert">
            Couldn&apos;t load hosts. Please try again.
          </div>
        )}
        {!hostsError && isLoadingHosts && (
          <div className={`${baseClass}__results-status`}>Loading...</div>
        )}
        {!hostsError && !isLoadingHosts && hosts.length === 0 && (
          <div className={`${baseClass}__results-status`}>
            No matching hosts.
          </div>
        )}
      </div>
    </div>
  );

  const renderHostFilters = () => (
    <div className={`${baseClass}__form`}>
      <Dropdown
        label="Labels"
        name="labels"
        options={labelOptions}
        value={selectedLabelIDs.join(",")}
        onChange={(value: string | null) => {
          if (!value) {
            setSelectedLabelIDs([]);
          } else {
            setSelectedLabelIDs(value.split(",").map(Number));
          }
        }}
        multi
        placeholder="All labels"
        searchable
        clearable
      />
      <Dropdown
        label="Platforms"
        name="platforms"
        options={PLATFORM_OPTIONS}
        value={selectedPlatforms.join(",")}
        onChange={(value: string | null) => {
          if (!value) {
            setSelectedPlatforms([]);
          } else {
            setSelectedPlatforms(value.split(","));
          }
        }}
        multi
        placeholder="All platforms"
        searchable={false}
        clearable
      />
      <TabNav secondary>
        <Tabs selectedIndex={tabIndex} onSelect={handleTabChange}>
          <TabList>
            <Tab>
              <TabText>Exclude hosts</TabText>
            </Tab>
            <Tab>
              <TabText>Specific hosts</TabText>
            </Tab>
          </TabList>
          {/* Only render the active tab to avoid two parallel host lists
              fighting over the shared listRef and duplicating API requests. */}
          <TabPanel>{tabIndex === 0 && renderHostSearch()}</TabPanel>
          <TabPanel>{tabIndex === 1 && renderHostSearch()}</TabPanel>
        </Tabs>
      </TabNav>
    </div>
  );

  return (
    <Modal title="Settings" onExit={onCancel} className={baseClass}>
      <form onSubmit={handleSubmit}>
        {isCVE ? (
          <TabNav>
            <Tabs selectedIndex={activeTab} onSelect={setActiveTab}>
              <TabList>
                <Tab>
                  <TabText>Hosts</TabText>
                </Tab>
                <Tab>
                  <TabText>Software</TabText>
                </Tab>
              </TabList>
              <TabPanel>{renderHostFilters()}</TabPanel>
              <TabPanel>
                {/* Wrap in __form so the Software tab gets the same bottom
                    spacing before the action buttons as the Hosts tab. */}
                <div className={`${baseClass}__form`}>
                  <SoftwareFilters
                    currentTeamId={currentTeamId}
                    categories={softwareFilters}
                    knownExploit={knownExploit}
                    epssMin={epssMin}
                    epssMax={epssMax}
                    severityFilter={severityFilter}
                    initialShowAdvanced={initialShowAdvanced}
                    errors={softwareErrors}
                    excludeCVEs={excludeCVEs}
                    setCategories={onChangeCategories}
                    setKnownExploit={setKnownExploit}
                    setEpssMin={onChangeEpssMin}
                    setEpssMax={onChangeEpssMax}
                    setSeverityFilter={onChangeSeverityFilter}
                    setExcludeCVEs={setExcludeCVEs}
                    onFieldBlur={onFieldBlur}
                    onFieldFocus={onFieldFocus}
                  />
                </div>
              </TabPanel>
            </Tabs>
          </TabNav>
        ) : (
          renderHostFilters()
        )}
        <div className={`${baseClass}__btn-wrap`}>
          {hasFilters && (
            <Button variant="secondary" onClick={handleClear}>
              Clear all
            </Button>
          )}
          <div className="modal-cta-wrap">
            {/* Always enabled: submitting with invalid input is what surfaces
            the inline errors. See
            frontend/docs/patterns.md#submit-button-state. */}
            <Button type="submit" variant="default">
              Apply
            </Button>
            <Button variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
};

export default ChartFilterModal;
