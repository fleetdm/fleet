import React from 'react';

import Icon from 'components/icons/Icon';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

const baseClass = 'pack-info-side-panel';

const PackInfoSidePanel = () => {
  return (
    <SecondarySidePanelContainer className={baseClass}>
      <h3 className={`${baseClass}__title`}>
        <Icon name="packs" />
        &nbsp;
        What&apos;s a Query Pack?
      </h3>
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

      <dl>
        <dt><Icon name="plus-minus" /> <span>Differential</span></dt>
        <dd>Only record data that has changed.</dd>

        <dt><Icon name="camera" /> <span>Snapshot</span></dt>
        <dd>Record full query result each time.</dd>
      </dl>

      <p>
        Packs are distributed to specified <b>targets</b>. Targets may be
        <b>individual hosts</b> or groups of hosts called <b>labels.</b>
      </p>
      <p>
        Learn more about Query Packs in the <a href="https://kolide.co">documentation</a>.
      </p>
    </SecondarySidePanelContainer>
  );
};

export default PackInfoSidePanel;
