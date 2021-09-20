import React from "react";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";

const baseClass = "empty-pack";

const EmptyPack = (): JSX.Element => {
  const searchEmpty = true;
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className={`${baseClass}__first-query`}>
          <h1>Your pack is empty.</h1>
          <span className={`${baseClass}__first-query-cta`}>
            Use the sidebar on the right to add queries to this pack.
          </span>
          <h1>Configure your queries.</h1>
          <p>
            <strong>Frequency:</strong> the amount of time, in seconds, the
            query waits before running
          </p>
          <p>
            <strong>Platform:</strong> the computer platform where this query
            will run (other platforms ignored)
          </p>
          <p>
            <strong>Minimum osquery version:</strong> the minimum required{" "}
            <strong>osqueryd</strong> version installed on a host
          </p>
          <p>
            <strong>Logging:</strong>
          </p>
          <ul>
            <li>
              <strong>
                <FleetIcon name="plus-minus" /> Differential:
              </strong>{" "}
              show only what’s added from last run
            </li>
            <li>
              <strong>
                <FleetIcon name="bold-plus" /> Differential (ignore removals):
              </strong>{" "}
              show only what’s been added since the last run
            </li>
            <li>
              <strong>
                <FleetIcon name="camera" /> Snapshot:
              </strong>{" "}
              show everything in its current state
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default EmptyPack;
