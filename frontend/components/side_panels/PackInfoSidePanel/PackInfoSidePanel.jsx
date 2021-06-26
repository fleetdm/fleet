import React from "react";

import SecondarySidePanelContainer from "../SecondarySidePanelContainer";
import DifferentialIcon from "../../../../assets/images/icon-plus-minus-black-16x16@2x.png";
import SnapshotIcon from "../../../../assets/images/icon-snapshot-black-16x14@2x.png";

const baseClass = "pack-info-side-panel";

const PackInfoSidePanel = () => {
  return (
    <SecondarySidePanelContainer className={baseClass}>
      <h3 className={`${baseClass}__title`}>What&apos;s a query pack?</h3>
      <p>
        Osquery supports grouping of queries (called query packs) which run on a
        scheduled basis and log the results to a configurable destination.
      </p>
      <p>
        Query Packs are useful for monitoring specific attributes of hosts over
        time and can be used for alerting and incident response investigations.
        By default, queries added to packs run every hour (interval = 3600s).
      </p>

      <p>Queries can be run in two modes:</p>

      <dl>
        <dt>
          <img src={DifferentialIcon} alt="plus-minus" />
          <span>Differential</span>
        </dt>

        <dt>
          <img src={SnapshotIcon} alt="snapshot" />
          <span>Snapshot</span>
        </dt>
      </dl>

      <h4 className={`${baseClass}__subtitle`}>Where do I find results?</h4>
      <p>
        Packs are distributed to specified targets. Targets may be individual
        hosts or groups of hosts called labels.
      </p>
      <p>
        The results of queries run via query packs are stored in log files for
        your convenience. We recommend forwarding these logs to a log
        aggregation tool or other actionable tool for further analysis. These
        logs can be found in the following locations:
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
