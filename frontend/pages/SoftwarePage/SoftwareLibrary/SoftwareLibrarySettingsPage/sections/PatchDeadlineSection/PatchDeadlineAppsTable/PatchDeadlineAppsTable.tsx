import React, { useEffect, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";

import List from "components/List";
import PageDescription from "components/PageDescription";
import Pagination from "components/Pagination";
import SectionHeader from "components/SectionHeader";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { IPolicySoftwareToInstall } from "interfaces/policy";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "patch-deadline-apps-table";

const PAGE_SIZE = 10;

// List requires each row to have a React.Key under some property. patch_software
// doesn't ship its own id, so project software_title_id in as `id` for keying.
type IAppRow = IPolicySoftwareToInstall & { id: number };

interface IPatchDeadlineAppsTableProps {
  // fleetId drives the page-reset effect and preserves fleet context on the
  // outgoing row link; the parent fetches policies by it too.
  fleetId: number;
  // Each app is the `patch_software` payload off a patch-automation policy.
  apps: IPolicySoftwareToInstall[];
  router: InjectedRouter;
}

const PatchDeadlineAppsTable = ({
  fleetId,
  apps,
  router,
}: IPatchDeadlineAppsTableProps) => {
  const [page, setPage] = useState(0);

  // Rewind pagination on fleet switch so a shorter list doesn't strand users
  // past its end. safePage clamps the render; this keeps the stored state in
  // sync so Next doesn't need extra clicks to re-engage.
  useEffect(() => {
    setPage(0);
  }, [fleetId]);

  // Alphanumeric + case-insensitive. `numeric: true` makes "App 2" sort before
  // "App 10". Client-side; revisit if lists grow large.
  const sortedApps: IAppRow[] = useMemo(
    () =>
      [...apps]
        .sort((a, b) =>
          getDisplayedSoftwareName(
            a.name,
            a.display_name
          ).localeCompare(
            getDisplayedSoftwareName(b.name, b.display_name),
            undefined,
            { numeric: true, sensitivity: "base" }
          )
        )
        .map((app) => ({ ...app, id: app.software_title_id })),
    [apps]
  );

  const totalPages = Math.max(1, Math.ceil(sortedApps.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages - 1);
  const pagedApps = sortedApps.slice(
    safePage * PAGE_SIZE,
    (safePage + 1) * PAGE_SIZE
  );

  const goToTitle = (app: IAppRow) => {
    router.push(
      getPathWithQueryParams(
        PATHS.SOFTWARE_TITLE_DETAILS(app.software_title_id.toString()),
        { fleet_id: fleetId }
      )
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader wrapperCustomClass={`${baseClass}__header`} title="Apps" />
      <PageDescription
        variant="tab-panel"
        content="Deadline options apply to these apps."
      />
      <List<IAppRow>
        data={pagedApps}
        heading={<span className={`${baseClass}__list-title`}>Name</span>}
        onClickRow={goToTitle}
        renderItemRow={(item) => {
          const displayName = getDisplayedSoftwareName(
            item.name,
            item.display_name
          );
          return (
            <div className={`${baseClass}__row-content`}>
              <SoftwareIcon name={item.name} url={item.icon_url ?? undefined} />
              <div className={`${baseClass}__name`}>
                <TooltipTruncatedText value={displayName} />
              </div>
            </div>
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
