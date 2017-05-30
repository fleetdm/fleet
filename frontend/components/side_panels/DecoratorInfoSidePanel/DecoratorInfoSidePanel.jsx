import React from 'react';

import Icon from 'components/icons/Icon';
import SecondarySidePanelContainer from '../SecondarySidePanelContainer';

const baseClass = 'decorator-info-side-panel';

const DecoratorInfoSidePanel = () => {
  return (
    <SecondarySidePanelContainer className={baseClass}>
      <h3 className={`${baseClass}__title`}>
        <Icon name="decorator" />
        &nbsp;
        What are Decorators?
      </h3>
      <p>
        Decorator queries are used to add additional information to results and snapshot logs.
        There are three types of decorator queries based on when and how you want to collect the decoration data.
      </p>
      <p>The types of decorators are:</p>
      <ul>
        <li>
          <strong>load:</strong> run these decorators when the configuration loads (or is reloaded).
        </li>
        <li>
          <strong>always:</strong> run these decorators before each query in the schedule.
        </li>
        <li>
          <strong>interval:</strong> run the decorator on a defined interval.  The interval must be a multiple of 60.
          If the interval period is not divisible by 60 validation will fail.
        </li>
      </ul>
      <p>
        Each decorator query should return at most 1 row. A warning will be generated if more than 1 row is returned as
        they will be forcefully ignored and constitute undefined behavior.
        Each decorator query should be careful not to emit column collisions, this is also undefined behavior.
      </p>
      <p>
        The command line flag decorators_top_level can be set to true to make decorator data populate as top
        level key/value objects instead of being contained as a child of decorations.
      </p>

    </SecondarySidePanelContainer>
  );
};

export default DecoratorInfoSidePanel;
