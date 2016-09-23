import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import { isEqual, last } from 'lodash';
import componentStyles from './styles';
import kolideLogo from '../../../assets/images/kolide-logo.svg';
import navItems from './navItems';
import './styles.scss';

class SidePanel extends Component {
  static propTypes = {
    user: PropTypes.object,
  };

  constructor (props) {
    super(props);

    this.state = {
      activeTab: 'Hosts',
      activeSubItem: 'Add Hosts',
      showSubItems: false,
    };
  }

  setActiveSubItem = (activeSubItem) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({ activeSubItem });
      return false;
    };
  }

  setActiveTab = (activeTab) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({
        activeTab,
      });

      return false;
    };
  }

  setSubNavClass = (showSubItems) => {
    return showSubItems ? 'sub-nav sub-nav--expanded' : 'sub-nav';
  }

  toggleShowSubItems = (showSubItems) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({ showSubItems });

      return false;
    };
  }

  renderHeader = () => {
    const {
      user: {
        enabled,
        username,
      },
    } = this.props;
    const {
      companyLogoStyles,
      headerStyles,
      orgNameStyles,
      usernameStyles,
      userStatusStyles,
      orgChevronStyles,
    } = componentStyles;

    return (
      <header style={headerStyles}>
        <img
          alt="Company logo"
          src={kolideLogo}
          style={companyLogoStyles}
        />
        <h1 style={orgNameStyles}>Kolide, Inc.</h1>
        <div style={userStatusStyles(enabled)} />
        <h2 style={usernameStyles}>{username}</h2>
        <i style={orgChevronStyles} className="kolidecon-chevron-bold-down" />
      </header>
    );
  }

  renderNavItem = (navItem, lastChild) => {
    const { activeTab } = this.state;
    const { icon, name, subItems } = navItem;
    const active = activeTab === name;
    const {
      iconStyles,
      navItemBeforeStyles,
      navItemNameStyles,
      navItemStyles,
      navItemWrapperStyles,
    } = componentStyles;
    const { renderSubItems, setActiveTab } = this;

    return (
      <div style={navItemWrapperStyles(lastChild)} key={`nav-item-${name}`}>
        {active && <div style={navItemBeforeStyles} />}
        <li
          key={name}
          onClick={setActiveTab(name)}
          style={navItemStyles(active)}
        >
          <div style={{ position: 'relative' }}>
            <i className={icon} style={iconStyles} />
            <span style={navItemNameStyles}>
              {name}
            </span>
          </div>
          {active && renderSubItems(subItems)}
        </li>
      </div>
    );
  }

  renderNavItems = () => {
    const { renderNavItem } = this;
    const { navItemListStyles } = componentStyles;
    const { user: { admin } } = this.props;
    const userNavItems = navItems(admin);

    return (
      <ul style={navItemListStyles}>
        {userNavItems.map((navItem, index, collection) => {
          const lastChild = admin && isEqual(navItem, last(collection));
          return renderNavItem(navItem, lastChild);
        })}
      </ul>
    );
  }

  renderSubItem = (subItem) => {
    const { activeSubItem } = this.state;
    const { name, path } = subItem;
    const active = activeSubItem === name;
    const { setActiveSubItem } = this;
    const {
      subItemBeforeStyles,
      subItemStyles,
      subItemLinkStyles,
    } = componentStyles;

    return (
      <div
        key={`sub-item-${name}`}
        style={{ position: 'relative' }}
      >
        {active && <div style={subItemBeforeStyles} />}
        <li
          key={name}
          onClick={setActiveSubItem(name)}
          style={subItemStyles(active)}
        >
          <span to={path} style={subItemLinkStyles(active)}>{name}</span>
        </li>
      </div>
    );
  }

  renderSubItems = (subItems) => {
    const { subItemListStyles, subItemsStyles } = componentStyles;
    const { renderCollapseSubItems, renderSubItem, setSubNavClass } = this;
    const { showSubItems } = this.state;

    return (
      <div className={setSubNavClass(showSubItems)} style={subItemsStyles}>
        <ul style={subItemListStyles(showSubItems)}>
          {subItems.map(subItem => {
            return renderSubItem(subItem);
          })}
        </ul>
        {renderCollapseSubItems()}
      </div>
    );
  }

  renderCollapseSubItems = () => {
    const { toggleShowSubItems } = this;
    const { showSubItems } = this.state;
    const { collapseSubItemsWrapper } = componentStyles;
    const iconName = showSubItems ? 'kolidecon-chevron-bold-left' : 'kolidecon-chevron-bold-right';

    return (
      <div style={collapseSubItemsWrapper} onClick={toggleShowSubItems(!showSubItems)}>
        <i className={iconName} />
      </div>
    );
  }

  render () {
    const { navStyles } = componentStyles;
    const { renderHeader, renderNavItems } = this;

    return (
      <nav style={navStyles}>
        {renderHeader()}
        {renderNavItems()}
      </nav>
    );
  }
}

export default radium(SidePanel);
