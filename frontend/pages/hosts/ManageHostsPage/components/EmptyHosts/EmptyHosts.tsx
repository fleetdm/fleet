/**
 * Component when there is no host results found in a search
 */
import React from 'react';

const baseClass = 'empty-hosts';

const EmptyHosts = (): JSX.Element => {
  return (
    <div className={`${baseClass}  ${baseClass}--no-hosts`}>
      <div className={`${baseClass}--no-hosts__inner`}>
        <div className={'no-filter-results'}>
          <h1>No hosts match the current criteria</h1>
          <p>Expecting to see new hosts? Try again in a few seconds as the system catches up</p>
        </div>
      </div>
    </div>
  );
};

export default EmptyHosts;
