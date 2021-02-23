import React, { Component } from 'react';
import PropTypes from 'prop-types';

import InputField from 'components/forms/fields/InputField';
import KolideIcon from 'components/icons/KolideIcon';

import HostsDataTable from '../HostsDataTable/HostsDataTable';

const baseClass = 'host-container';

// TODO: come back and define table columns here

class HostContainer extends Component {
  static propTypes = {
    selectedFilter: PropTypes.string,
  };

  constructor (props) {
    super(props);

    this.state = {
      searchQuery: '',
    };
  }

  onSearchQueryChange = (newQuery) => {
    this.setState({
      searchQuery: newQuery,
    });
  }

  render () {
    const { onSearchQueryChange } = this;
    const { selectedFilter } = this.props;
    const { searchQuery } = this.state;
    console.log(selectedFilter);


    // if (hosts.length === 0) {
    //   if (selectedLabel && selectedLabel.type === 'all') {
    //     return (
    //       <div className={`${baseClass}  ${baseClass}--no-hosts`}>
    //         <div className={`${baseClass}--no-hosts__inner`}>
    //           <img src={RoboDogImage} alt="No Hosts" />
    //           <div>
    //             <h1>It&#39;s kinda empty in here...</h1>
    //             <h2>Get started adding hosts to Fleet.</h2>
    //             <p>Add your laptops and servers to securely monitor them.</p>
    //             <div className={`${baseClass}__no-hosts-contact`}>
    //               <p>Still having trouble?</p>
    //               <a href="https://github.com/fleetdm/fleet/issues">File a GitHub issue</a>
    //             </div>
    //           </div>
    //         </div>
    //       </div>
    //     );
    //   }

    //   return renderNoHosts();
    // }

    return (
      <div className={`${baseClass}`}>
        <div className={`${baseClass}__search-input`}>
          <InputField
            placeholder="Search hosts by hostname"
            name=""
            onChange={onSearchQueryChange}
            value={searchQuery}
            inputWrapperClass={`${baseClass}__input-wrapper`}
          />
          <KolideIcon name="search" />
        </div>
        <HostsDataTable
          selectedFilter={selectedFilter}
          searchQuery={searchQuery}
        />
      </div>
    );
  }
}

export default HostContainer;
