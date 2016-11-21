import React from 'react';

import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

const classBlock = 'pack-info-side-panel';

class PackInfoSidePanel extends React.Component {
  render () {
    return (
      <SecondarySidePanelContainer className={classBlock}>
        <div>
          <h3>
            <i className="kolidecon-packs" />
            &nbsp;
            What&apos;s a Query Pack?
          </h3>
        </div>
        <hr />
        <p>
          Osquery supports grouping of queries (called <b>query packs</b>)
          which run on a scheduled basis and log the results to a configurable
          destination.
        </p>
        <p>
          Query Packs are useful for monitoring specific attributes of hosts
          over time and can be used for alerting and incident response
          investigations. By default, queries added to packs run every hour
          (<b>interval = 3600s</b>).
        </p>
        <p>
          Queries can be run in two modes:
        </p>
        <p>
          <b>-Differential:</b>
        </p>
        <p>
          Only record data that has changed.
        </p>
        <p>
          <b>-Snapshot:</b>
        </p>
        <p>
          Record full query result each time.
        </p>
        <p>
          Packs are distributed to specified <b>targets</b>. Targets may be
          <b>individual hosts</b> or groups of hosts called <b>labels.</b>
        </p>
        <p>
          Learn more about Query Packs in the
          <a href="https://kolide.co">
              documentation
          </a>.
        </p>
      </SecondarySidePanelContainer>
    );
  }
}

export default PackInfoSidePanel;
