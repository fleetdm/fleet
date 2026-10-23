import React, { useEffect, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";

import PageDescription from "components/PageDescription";
import Pagination from "components/Pagination";
import SectionHeader from "components/SectionHeader";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import UploadList from "components/UploadList";
import { IPolicySoftwareToInstall } from "interfaces/policy";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "patch-deadline-apps-table";

const PAGE_SIZE = 10;

interface IPatchDeadlineAppsTableProps {
  // fleetId drives the page-reset effect and preserves fleet context on the
  // outgoing row link; the parent fetches policies by it too.
  fleetId: number;
  // Each app is the `install_software` payload off a patch-automation policy.
  // No id of its own — the row key uses software_title_id.
  apps: IPolicySoftwareToInstall[];
  router: InjectedRouter;
}

const PatchDeadlineAppsTable = ({
  fleetId,
  apps,
  router,
}: IPatchDeadlineAppsTableProps) => {
  const [page, setPage] = useState(0);

  // Switching fleets rewinds pagination so the user doesn't land past the end
  // of a shorter list. (safePage clamps the render; this keeps the stored
  // state in sync so Next doesn't require multiple clicks to re-engage.)
  useEffect(() => {
    setPage(0);
  }, [fleetId]);

  // Alphanumeric, case-insensitive sort on the admin-visible name. `numeric`
  // makes "App 2" sort before "App 10"; `sensitivity: "base"` folds case so
  // mixed-case titles don't clump by capitalization. Client-side is fine while
  // the list is small; revisit if a fleet's deadline set grows large enough to
  // warrant server sort/paginate.
  const sortedApps = useMemo(
    () =>
      [...apps].sort((a, b) =>
        getDisplayedSoftwareName(
          a.name,
          a.display_name
        ).localeCompare(
          getDisplayedSoftwareName(b.name, b.display_name),
          undefined,
          { numeric: true, sensitivity: "base" }
        )
      ),
    [apps]
  );

  const totalPages = Math.max(1, Math.ceil(sortedApps.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages - 1);
  const pagedApps = sortedApps.slice(
    safePage * PAGE_SIZE,
    (safePage + 1) * PAGE_SIZE
  );

  const goToTitle = (id: number) => {
    router.push(
      getPathWithQueryParams(PATHS.SOFTWARE_TITLE_DETAILS(id.toString()), {
        fleet_id: fleetId,
      })
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader wrapperCustomClass={`${baseClass}__header`} title="Apps" />
      <PageDescription
        variant="tab-panel"
        content="Deadline options apply to these apps."
      />
      <UploadList<IPolicySoftwareToInstall>
        className={`${baseClass}__list`}
        keyAttribute="software_title_id"
        listItems={pagedApps}
        HeadingComponent={() => (
          <span className={`${baseClass}__list-title`}>Name</span>
        )}
        ListItemComponent={({ listItem }) => {
          const displayName = getDisplayedSoftwareName(
            listItem.name,
            listItem.display_name
          );
          return (
            <button
              type="button"
              className={`${baseClass}__row`}
              onClick={() => goToTitle(listItem.software_title_id)}
            >
              <SoftwareIcon
                name={listItem.name}
                url={listItem.icon_url ?? undefined}
              />
              <div className={`${baseClass}__name`}>
                <TooltipTruncatedText value={displayName} />
              </div>
            </button>
          );
        }}
      />
      <Pagination
        disablePrev={safePage === 0}
        disableNext={safePage >= totalPages - 1}
        onPrevPage={() => setPage((p) => Math.max(0, p - 1))}
        onNextPage={() => setPage((p) => p + 1)}
        hidePagination={sortedApps.length <= PAGE_SIZE}
      />
    </div>
  );
};

export default PatchDeadlineAppsTable;
