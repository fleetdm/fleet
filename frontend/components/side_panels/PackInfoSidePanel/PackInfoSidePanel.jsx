import React from "react";

import KolideIcon from "components/icons/KolideIcon";
import SecondarySidePanelContainer from "../SecondarySidePanelContainer";

const baseClass = "pack-info-side-panel";

const PackInfoSidePanel = () => {
  return (
    <SecondarySidePanelContainer className={baseClass}>
      <h3 className={`${baseClass}__title`}>
        <KolideIcon name="packs" />
        &nbsp; What&apos;s a query pack?
      </h3>
      <p>
        Osquery supports grouping of queries (called <b>query packs</b>) which
        run on a scheduled basis and log the results to a configurable
        destination.
      </p>
      <p>
        Query Packs are useful for monitoring specific attributes of hosts over
        time and can be used for alerting and incident response investigations.
        By default, queries added to packs run every hour (
        <b>interval = 3600s</b>).
      </p>

      <p>Queries can be run in two modes:</p>

      <dl>
        <dt>
          <KolideIcon name="plus-minus" /> <span>Differential</span>
        </dt>
        <dd>Only record data that has changed.</dd>

        <dt>
          <KolideIcon name="camera" /> <span>Snapshot</span>
        </dt>
        <dd>Record full query result each time.</dd>
      </dl>

      <h4 className={`${baseClass}__subtitle`}>Where do I find results?</h4>
      <p>
        Packs are distributed to specified <b>targets</b>. Targets may be{" "}
        <b>individual hosts</b> or groups of hosts called <b>labels.</b>
      </p>
      <p>
        The results of queries run via query packs are stored in log files for
        your convenience. We recommend forwarding this logs to a log aggregation
        tool or other actionable tool for further analysis. These logs can be
        found in the following locations:
      </p>
      <ul>
        <li>
          <strong>Status Log:</strong> /path/to/status/logs
        </li>
        <li>
          <strong>Result Log:</strong> /path/to/result/logs
        </li>
      </ul>
      <p>
        Learn more about log aggregation in the{" "}
        <a
          href="https://osquery.readthedocs.io/en/stable/deployment/log-aggregation/"
          target="_blank"
          rel="noopener noreferrer"
        >
          documentation
        </a>
        .
      </p>
    </SecondarySidePanelContainer>
  );
};

export default PackInfoSidePanel;
