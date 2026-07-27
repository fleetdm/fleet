import React, { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce";

import { IVulnerability } from "interfaces/vulnerability";
import { CVE_SOFTWARE_CATEGORIES } from "interfaces/charts";
import {
  getVulnerabilities,
  IVulnerabilitiesResponse,
} from "services/entities/vulnerabilities";

import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import RevealButton from "components/buttons/RevealButton";
import SearchField from "components/forms/fields/SearchField";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import { getEpssError, NO_CATEGORIES_MSG } from "./helpers";

const baseClass = "software-filters";

const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 300;

interface ISoftwareFiltersProps {
  currentTeamId?: number;
  categories: string[];
  knownExploit: boolean;
  epssMin: string;
  epssMax: string;
  excludeCVEs: string[];
  setCategories: (categories: string[]) => void;
  setKnownExploit: (value: boolean) => void;
  setEpssMin: (value: string) => void;
  setEpssMax: (value: string) => void;
  setExcludeCVEs: (cves: string[]) => void;
}

const SoftwareFilters = ({
  currentTeamId,
  categories,
  knownExploit,
  epssMin,
  epssMax,
  excludeCVEs,
  setCategories,
  setKnownExploit,
  setEpssMin,
  setEpssMax,
  setExcludeCVEs,
}: ISoftwareFiltersProps): JSX.Element => {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [pageCount, setPageCount] = useState(1);
  const listRef = useRef<HTMLDivElement>(null);

  const excludedSet = new Set(excludeCVEs);

  const debouncedSetSearchQuery = useDebouncedCallback((value: string) => {
    setSearchQuery(value);
    setPageCount(1);
    if (listRef.current) {
      listRef.current.scrollTop = 0;
    }
  }, SEARCH_DEBOUNCE_MS);

  useEffect(() => {
    return () => debouncedSetSearchQuery.cancel();
  }, [debouncedSetSearchQuery]);

  // Search all CVEs (not just the curated set) so a user can exclude anything.
  // Mirrors the host search: keep prior pages cached and grow per_page on scroll.
  const {
    data: vulnData,
    isLoading: isLoadingVulns,
    error: vulnsError,
  } = useQuery<IVulnerabilitiesResponse, Error>(
    ["chartFilterCVEs", currentTeamId, searchQuery, pageCount],
    () =>
      getVulnerabilities({
        teamId: currentTeamId,
        page: 0,
        per_page: pageCount * PAGE_SIZE,
        query: searchQuery || undefined,
      }),
    // The CVE search UI lives entirely inside the Advanced section, so don't
    // fetch until the user opens it.
    { keepPreviousData: true, staleTime: 30000, enabled: showAdvanced }
  );

  const cves: IVulnerability[] = vulnData?.vulnerabilities ?? [];
  const hasMore = vulnData?.meta?.has_next_results ?? false;

  const handleScroll = useCallback(() => {
    const el = listRef.current;
    if (!el || !hasMore || isLoadingVulns) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 40) {
      setPageCount((prev) => prev + 1);
    }
  }, [hasMore, isLoadingVulns]);

  const handleSearchChange = useCallback(
    (value: string) => {
      setSearchInput(value);
      debouncedSetSearchQuery(value);
    },
    [debouncedSetSearchQuery]
  );

  const toggleCategory = (value: string) => {
    if (categories.includes(value)) {
      setCategories(categories.filter((c) => c !== value));
    } else {
      setCategories([...categories, value]);
    }
  };

  const toggleCVE = (cve: string) => {
    if (excludedSet.has(cve)) {
      setExcludeCVEs(excludeCVEs.filter((c) => c !== cve));
    } else {
      setExcludeCVEs([...excludeCVEs, cve]);
    }
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__categories`}>
        {CVE_SOFTWARE_CATEGORIES.map((cat) => (
          <Checkbox
            key={cat.value}
            name={`category-${cat.value}`}
            value={categories.includes(cat.value)}
            onChange={() => toggleCategory(cat.value)}
            helpText={cat.description || undefined}
          >
            {cat.label}
          </Checkbox>
        ))}
        {categories.length === 0 && (
          <div className={`${baseClass}__categories-error`} role="alert">
            {NO_CATEGORIES_MSG}
          </div>
        )}
      </div>

      <div className={`${baseClass}__kev`}>
        <h3 className={`${baseClass}__section-title`}>
          CISA known exploit (KEV)
        </h3>
        <Checkbox
          name="known-exploit"
          value={knownExploit}
          onChange={() => setKnownExploit(!knownExploit)}
          helpText="Software has vulnerabilities that have been actively exploited in the wild."
        >
          Has known exploit
        </Checkbox>
      </div>

      <RevealButton
        className={`${baseClass}__advanced-toggle`}
        isShowing={showAdvanced}
        showText="Advanced options"
        hideText="Advanced options"
        caretPosition="after"
        onClick={() => setShowAdvanced((prev) => !prev)}
        variant="secondary"
      />

      {showAdvanced && (
        <div className={`${baseClass}__advanced`}>
          <div className={`${baseClass}__epss`}>
            <h3 className={`${baseClass}__section-title`}>
              <TooltipWrapper
                tooltipClass={`${baseClass}__tooltip-text`}
                tipContent={
                  <>
                    The probability that this vulnerability will be exploited in
                    the next 30 days (EPSS probability). <br />
                    This data is reported by FIRST.org.
                  </>
                }
              >
                Probability of exploit
              </TooltipWrapper>
            </h3>
            <p className={`${baseClass}__section-help`}>
              EPSS probabilities range from 0 to 100%.
            </p>
            <div className={`${baseClass}__epss-inputs`}>
              <InputField
                label="Min"
                name="epss-min"
                type="number"
                value={epssMin}
                placeholder="0"
                error={getEpssError(epssMin)}
                onChange={setEpssMin}
              />
              <InputField
                label="Max"
                name="epss-max"
                type="number"
                value={epssMax}
                placeholder="100"
                error={getEpssError(epssMax)}
                onChange={setEpssMax}
              />
            </div>
          </div>

          <div className={`${baseClass}__exclude-cves`}>
            <h3 className={`${baseClass}__section-title`}>
              Exclude vulnerabilities (CVEs)
            </h3>
            <SearchField
              placeholder="Search CVEs"
              defaultValue={searchInput}
              onChange={handleSearchChange}
            />
            {excludeCVEs.length > 0 && (
              <div className={`${baseClass}__pills`}>
                {excludeCVEs.map((cve) => (
                  <button
                    key={cve}
                    type="button"
                    className={`${baseClass}__pill`}
                    onClick={() => toggleCVE(cve)}
                  >
                    {cve}
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
              {cves.map((vuln) => (
                <div key={vuln.cve} className={`${baseClass}__results-row`}>
                  <Checkbox
                    name={`cve-${vuln.cve}`}
                    value={excludedSet.has(vuln.cve)}
                    onChange={() => toggleCVE(vuln.cve)}
                  >
                    {vuln.cve}
                  </Checkbox>
                </div>
              ))}
              {vulnsError && (
                <div className={`${baseClass}__results-status`} role="alert">
                  Couldn&apos;t load CVEs. Please try again.
                </div>
              )}
              {!vulnsError && isLoadingVulns && (
                <div className={`${baseClass}__results-status`}>Loading...</div>
              )}
              {!vulnsError && !isLoadingVulns && cves.length === 0 && (
                <div className={`${baseClass}__results-status`}>
                  No matching CVEs.
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SoftwareFilters;
